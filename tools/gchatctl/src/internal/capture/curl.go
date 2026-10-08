package capture

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

func parseCURL(input string) (Request, error) {
	args, err := tokenize(input)
	if err != nil {
		return Request{}, errors.New("invalid cURL quoting")
	}
	if len(args) == 0 || args[0] != "curl" {
		return Request{}, errors.New("input is not a cURL command")
	}
	method := "GET"
	var rawURL string
	var pairs [][2]string
	var cookies []string
	hasBody := false
	body := ""
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://") {
			if rawURL != "" {
				return Request{}, errors.New("cURL must contain exactly one request URL")
			}
			rawURL = arg
			continue
		}
		value, consumed := optionValue(args, i, "-X", "--request")
		if consumed {
			method = value
			i++
			continue
		}
		value, consumed = optionValue(args, i, "-b", "--cookie")
		if consumed {
			cookies = append(cookies, value)
			i++
			continue
		}
		value, consumed = optionValue(args, i, "-H", "--header")
		if consumed {
			if key, val, ok := strings.Cut(value, ":"); ok {
				pairs = append(pairs, [2]string{key, val})
			}
			i++
			continue
		}
		for _, flag := range []string{"--data", "--data-raw", "--data-binary", "--data-urlencode", "-d", "-F", "--form"} {
			if value, consumed = optionValue(args, i, flag); consumed {
				hasBody = true
				body = value
				if method == "GET" {
					method = "POST"
				}
				i++
				break
			}
		}
	}
	if rawURL == "" {
		return Request{}, errors.New("cURL must contain exactly one request URL")
	}
	if _, err := url.Parse(rawURL); err != nil {
		return Request{}, errors.New("invalid request URL")
	}
	headers := headerMap(pairs)
	return buildRequest(method, rawURL, headers, cookies, queryNamesFromURL(rawURL), hasBody, 200, true, body)
}

func optionValue(args []string, i int, names ...string) (string, bool) {
	for _, name := range names {
		if args[i] == name && i+1 < len(args) {
			return args[i+1], true
		}
		if strings.HasPrefix(args[i], name+"=") {
			return strings.TrimPrefix(args[i], name+"="), true
		}
	}
	return "", false
}

func tokenize(input string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	escaped := false
	started := false
	flush := func() {
		if started {
			args = append(args, word.String())
			word.Reset()
			started = false
		}
	}
	runes := []rune(input)
	for i, r := range runes {
		if escaped {
			if r != '\n' {
				word.WriteRune(r)
				started = true
			}
			escaped = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			started = true
			continue
		}
		if r == '$' && i+1 < len(runes) && runes[i+1] == '\'' {
			started = true
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
		} else if unicode.IsSpace(r) {
			flush()
		} else {
			word.WriteRune(r)
			started = true
		}
		if i == len(runes)-1 {
			flush()
		}
	}
	if quote != 0 || escaped {
		return nil, errors.New("unterminated shell token")
	}
	flush()
	return args, nil
}
