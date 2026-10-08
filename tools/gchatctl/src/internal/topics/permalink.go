package topics

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var chatIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,22}$`)

type Permalink struct {
	SpaceID   string
	ThreadID  string
	MessageID string
}

func (p Permalink) StartWebID() string {
	if p.MessageID != "" {
		return p.MessageID
	}
	return p.ThreadID
}

func ParsePermalink(input string) (Permalink, error) {
	parsed, err := url.Parse(input)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Host != "chat.google.com" || parsed.Host != parsed.Hostname() {
		return Permalink{}, errors.New("conversation must be a https://chat.google.com/room/... permalink")
	}
	if parsed.RawPath != "" || strings.Contains(parsed.Path, "//") || strings.HasSuffix(parsed.Path, "/") {
		return Permalink{}, errors.New("malformed Google Chat permalink")
	}
	parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(parts) < 2 || len(parts) > 4 || parts[0] != "room" || !chatIDPattern.MatchString(parts[1]) {
		return Permalink{}, errors.New("malformed Google Chat permalink")
	}
	ref := Permalink{SpaceID: parts[1]}
	if len(parts) >= 3 {
		if !chatIDPattern.MatchString(parts[2]) {
			return Permalink{}, errors.New("malformed Google Chat permalink")
		}
		ref.ThreadID = parts[2]
	}
	if len(parts) == 4 {
		if !chatIDPattern.MatchString(parts[3]) {
			return Permalink{}, errors.New("malformed Google Chat permalink")
		}
		ref.MessageID = parts[3]
	}
	if ref.ThreadID == "" {
		return Permalink{}, errors.New("permalink must include a thread or message ID as the inclusive starting point")
	}
	if ref.MessageID == "" {
		ref.MessageID = ref.ThreadID
	}
	return ref, nil
}
