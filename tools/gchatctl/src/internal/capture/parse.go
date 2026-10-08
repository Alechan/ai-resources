package capture

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/url"
	"slices"
	"strings"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
)

const maxCURLInput = 16 << 20

var extraHeaderAllowlist = map[string]bool{
	"authorization":          true,
	"origin":                 true,
	"referer":                true,
	"x-framework-xsrf-token": true,
	"x-goog-authuser":        true,
	"x-goog-chat-space-id":   true,
	"x-goog-pageid":          true,
	"x-origin":               true,
	"x-requested-with":       true,
}

type Request struct {
	AccountKey    string
	RequestHost   string
	Method        string
	URL           string
	URLPath       string
	QueryNames    []string
	HeaderNames   []string
	Cookie        auth.Secret
	Authorization auth.Secret
	ExtraHeaders  map[string]auth.Secret
	HasBody       bool
	Body          auth.Secret
	RawQuery      string
	ContentType   string
	Status        int
}

func (r Request) Template(operation string) auth.Template {
	return auth.Template{
		Operation:   operation,
		Method:      r.Method,
		URLPath:     r.URLPath,
		RawQuery:    r.RawQuery,
		ContentType: r.ContentType,
		QueryNames:  slices.Clone(r.QueryNames),
		HeaderNames: filterHeaderNames(r.HeaderNames),
		HasBody:     r.HasBody,
		Body:        r.Body.Reveal(),
	}
}

func Parse(input string) (Request, error) {
	requests, err := ParseAll(input)
	if err != nil {
		return Request{}, err
	}
	return Select(requests)
}

func Select(requests []Request) (Request, error) {
	return selectRequest(requests)
}

func SelectMatching(requests []Request, needle string) (Request, error) {
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" {
		return selectRequest(requests)
	}
	needles := []string{needle}
	if needle == "search.messages" {
		needles = append(needles, "batchexecute", "sbnmjb")
	}
	var matches []Request
	for _, request := range requests {
		haystack := strings.ToLower(request.URLPath + " " + request.RawQuery + " " + request.URL)
		for _, item := range needles {
			if strings.Contains(haystack, item) {
				matches = append(matches, request)
				break
			}
		}
	}
	return selectRequest(matches)
}

func ParseAll(input string) ([]Request, error) {
	return ParseAllReader(strings.NewReader(input))
}

func ParseAllReader(reader io.Reader) ([]Request, error) {
	buffered := bufio.NewReader(reader)
	if err := skipBOMAndWhitespace(buffered); err != nil {
		return nil, err
	}
	preview, err := buffered.Peek(1)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("capture input is empty")
		}
		return nil, errors.New("could not read capture input")
	}
	if preview[0] == '{' {
		return parseHARReader(buffered)
	}
	data, err := io.ReadAll(io.LimitReader(buffered, maxCURLInput+1))
	if err != nil || len(data) > maxCURLInput {
		clear(data)
		return nil, errors.New("could not read capture input")
	}
	request, parseErr := parseCURL(string(data))
	clear(data)
	if parseErr != nil {
		return nil, parseErr
	}
	return []Request{request}, nil
}

func skipBOMAndWhitespace(reader *bufio.Reader) error {
	header, err := reader.Peek(3)
	if err != nil && !errors.Is(err, io.EOF) {
		return errors.New("could not read capture input")
	}
	if len(header) >= 3 && bytes.Equal(header, []byte{0xef, 0xbb, 0xbf}) {
		if _, err := reader.Discard(3); err != nil {
			return errors.New("could not read capture input")
		}
	}
	for {
		next, err := reader.Peek(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return errors.New("capture input is empty")
			}
			return errors.New("could not read capture input")
		}
		switch next[0] {
		case ' ', '\n', '\r', '\t':
			if _, err := reader.Discard(1); err != nil {
				return errors.New("could not read capture input")
			}
		default:
			return nil
		}
	}
}

func selectRequest(requests []Request) (Request, error) {
	if len(requests) == 0 {
		return Request{}, errors.New("no successful Google Chat request was found")
	}
	chosen := requests[0]
	best := requestScore(chosen)
	for _, request := range requests[1:] {
		if score := requestScore(request); score > best {
			chosen = request
			best = score
		}
	}
	return chosen, nil
}

func requestScore(request Request) int {
	score := (3-hostRank(request.RequestHost))*10 + (1 - pathRank(request.URLPath))
	if request.Cookie.Reveal() != "" {
		score += 100
	}
	return score
}

func hostRank(host string) int {
	switch host {
	case "chat.google.com":
		return 0
	case "clients6.google.com":
		return 1
	case "chat.googleapis.com":
		return 2
	default:
		return 3
	}
}

func buildRequest(method, rawURL string, headers map[string]string, cookies []string, queryNames []string, hasBody bool, status int, requireCookie bool, body string) (Request, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return Request{}, errors.New("invalid request URL")
	}
	host := strings.ToLower(parsed.Hostname())
	if !AllowedHost(host) {
		return Request{}, errors.New("request host is not a Google Chat host")
	}
	cookie := headers["cookie"]
	if cookie == "" && len(cookies) > 0 {
		cookie = strings.Join(cookies, "; ")
	}
	if cookie == "" && requireCookie {
		return Request{}, errors.New("request cookie was not found")
	}
	headerNames := make([]string, 0, len(headers)+1)
	for name := range headers {
		headerNames = append(headerNames, name)
	}
	if cookie != "" {
		if _, ok := headers["cookie"]; !ok {
			headerNames = append(headerNames, "cookie")
		}
	}
	slices.Sort(headerNames)
	sortedQuery := slices.Clone(queryNames)
	slices.Sort(sortedQuery)
	sortedQuery = slices.Compact(sortedQuery)
	extra := map[string]auth.Secret{}
	for name, value := range headers {
		if extraHeaderAllowlist[name] && name != "authorization" && value != "" {
			extra[name] = auth.NewSecret(value)
		}
	}
	if len(extra) == 0 {
		extra = nil
	}
	return Request{
		AccountKey:    AccountKey(rawURL, headers["x-goog-authuser"], ""),
		RequestHost:   host,
		Method:        strings.ToUpper(method),
		URL:           rawURL,
		URLPath:       NormalizePath(rawURL),
		QueryNames:    sortedQuery,
		HeaderNames:   headerNames,
		Cookie:        auth.NewSecret(cookie),
		Authorization: auth.NewSecret(headers["authorization"]),
		ExtraHeaders:  extra,
		HasBody:       hasBody || body != "",
		Body:          auth.NewSecret(body),
		RawQuery:      parsed.RawQuery,
		ContentType:   headers["content-type"],
		Status:        status,
	}, nil
}

func headerMap(pairs [][2]string) map[string]string {
	headers := map[string]string{}
	for _, pair := range pairs {
		name := strings.ToLower(strings.TrimSpace(pair[0]))
		value := strings.TrimSpace(pair[1])
		if name == "" {
			continue
		}
		headers[name] = value
	}
	return headers
}

func queryNamesFromURL(rawURL string) []string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(parsed.Query()))
	for name := range parsed.Query() {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
