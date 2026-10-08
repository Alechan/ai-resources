package curlparse

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
)

type Result struct {
	WorkspaceHost string
	Token         auth.Secret
	Cookie        auth.Secret
}

var multipartTokenPattern = regexp.MustCompile(`(?is)name="token"[^\r\n]*(?:\r?\n){2}([^\r\n]+)`)

func Parse(input string) (Result, error) {
	args, err := tokenize(input)
	if err != nil {
		return Result{}, errors.New("invalid cURL quoting")
	}
	if len(args) == 0 || args[0] != "curl" {
		return Result{}, errors.New("input is not a cURL command")
	}
	var hosts []string
	var cookie, token string
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://") {
			u, parseErr := url.Parse(arg)
			if parseErr != nil || u.Hostname() == "" {
				return Result{}, errors.New("invalid request URL")
			}
			hosts = append(hosts, strings.ToLower(u.Hostname()))
			continue
		}
		value, consumed := optionValue(args, i, "-b", "--cookie")
		if consumed {
			cookie = value
			i++
			continue
		}
		value, consumed = optionValue(args, i, "-H", "--header")
		if consumed {
			if key, val, ok := strings.Cut(value, ":"); ok && strings.EqualFold(strings.TrimSpace(key), "cookie") {
				cookie = strings.TrimSpace(val)
			}
			i++
			continue
		}
		for _, flag := range []string{"--data", "--data-raw", "--data-urlencode", "-d", "-F", "--form"} {
			if value, consumed = optionValue(args, i, flag); consumed {
				if parsed := extractToken(value); parsed != "" {
					token = parsed
				}
				i++
				break
			}
		}
	}
	if len(hosts) != 1 {
		return Result{}, errors.New("cURL must contain exactly one request host")
	}
	host := hosts[0]
	if host != "slack.com" && !strings.HasSuffix(host, ".slack.com") {
		return Result{}, errors.New("request host is not a Slack host")
	}
	if token == "" {
		return Result{}, errors.New("Slack token was not found")
	}
	if cookie == "" {
		return Result{}, errors.New("Slack cookie was not found")
	}
	return Result{WorkspaceHost: host, Token: auth.NewSecret(token), Cookie: auth.NewSecret(cookie)}, nil
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

func extractToken(body string) string {
	body = strings.TrimPrefix(body, "?")
	for _, field := range strings.Split(body, "&") {
		key, value, ok := strings.Cut(field, "=")
		if ok && key == "token" {
			decoded, err := url.QueryUnescape(value)
			if err == nil {
				return decoded
			}
		}
	}
	normalized := strings.NewReplacer(`\r`, "\r", `\n`, "\n").Replace(body)
	if match := multipartTokenPattern.FindStringSubmatch(normalized); len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
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
