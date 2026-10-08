package capture

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
)

type harEntry struct {
	Request  harRequest  `json:"request"`
	Response harResponse `json:"response"`
}

type harRequest struct {
	Method      string         `json:"method"`
	URL         string         `json:"url"`
	Headers     []harNameValue `json:"headers"`
	QueryString []harNameValue `json:"queryString"`
	Cookies     []harNameValue `json:"cookies"`
	PostData    *harPostData   `json:"postData"`
}

type harResponse struct {
	Status int `json:"status"`
}

type harNameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harPostData struct {
	hasText bool
	text    string
}

var errSkipHAR = errors.New("skip har entry")

func (p *harPostData) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		p.hasText = true
		return nil
	}
	if text, ok := raw["text"]; ok && string(text) != `""` && string(text) != "null" {
		p.hasText = true
		_ = json.Unmarshal(text, &p.text)
	}
	if params, ok := raw["params"]; ok && string(params) != "null" && string(params) != "[]" {
		p.hasText = true
	}
	return nil
}

func parseHAR(input string) ([]Request, error) {
	return parseHARReader(strings.NewReader(input))
}

func parseHARReader(reader io.Reader) ([]Request, error) {
	decoder := json.NewDecoder(reader)
	if err := skipToHAREntries(decoder); err != nil {
		return nil, errors.New("input is not a HAR document")
	}
	var requests []Request
	var firstErr error
	for decoder.More() {
		var entry harEntry
		if err := decoder.Decode(&entry); err != nil {
			return nil, errors.New("input is not a HAR document")
		}
		request, err := requestFromHAREntry(entry)
		if err != nil {
			if errors.Is(err, errSkipHAR) {
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		requests = append(requests, request)
	}
	if len(requests) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, errors.New("no successful Google Chat request was found")
	}
	return requests, nil
}

func requestFromHAREntry(entry harEntry) (Request, error) {
	if entry.Response.Status < 200 || entry.Response.Status > 299 {
		return Request{}, errSkipHAR
	}
	parsed, err := url.Parse(entry.Request.URL)
	if err != nil || parsed.Hostname() == "" {
		return Request{}, errors.New("invalid request URL")
	}
	if !AllowedHost(parsed.Hostname()) || IsStaticPath(parsed.Path) {
		return Request{}, errSkipHAR
	}
	headers := headerMap(nameValues(entry.Request.Headers))
	queryNames := make([]string, 0, len(entry.Request.QueryString))
	for _, item := range entry.Request.QueryString {
		if item.Name != "" {
			queryNames = append(queryNames, item.Name)
		}
	}
	if len(queryNames) == 0 {
		queryNames = queryNamesFromURL(entry.Request.URL)
	}
	cookies := make([]string, 0, len(entry.Request.Cookies))
	for _, item := range entry.Request.Cookies {
		if item.Name == "" {
			continue
		}
		cookies = append(cookies, item.Name+"="+item.Value)
	}
	hasBody := entry.Request.PostData != nil && entry.Request.PostData.hasText
	body := ""
	if entry.Request.PostData != nil {
		body = entry.Request.PostData.text
	}
	return buildRequest(entry.Request.Method, entry.Request.URL, headers, cookies, queryNames, hasBody, entry.Response.Status, false, body)
}

func skipToHAREntries(decoder *json.Decoder) error {
	if err := expectDelim(decoder, '{'); err != nil {
		return err
	}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return err
		}
		name, _ := key.(string)
		if name == "log" {
			return skipToLogEntries(decoder)
		}
		if err := skipValue(decoder); err != nil {
			return err
		}
	}
	return errors.New("input is not a HAR document")
}

func skipToLogEntries(decoder *json.Decoder) error {
	if err := expectDelim(decoder, '{'); err != nil {
		return err
	}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return err
		}
		name, _ := key.(string)
		if name == "entries" {
			return expectDelim(decoder, '[')
		}
		if err := skipValue(decoder); err != nil {
			return err
		}
	}
	return errors.New("input is not a HAR document")
}

func expectDelim(decoder *json.Decoder, want json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	got, ok := token.(json.Delim)
	if !ok || got != want {
		return errors.New("input is not a HAR document")
	}
	return nil
}

func skipValue(decoder *json.Decoder) error {
	var unused json.RawMessage
	return decoder.Decode(&unused)
}

func nameValues(items []harNameValue) [][2]string {
	pairs := make([][2]string, 0, len(items))
	for _, item := range items {
		pairs = append(pairs, [2]string{item.Name, item.Value})
	}
	return pairs
}
