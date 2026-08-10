package export

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	workspaceIDPattern    = regexp.MustCompile(`^T[A-Z0-9]{8,}$`)
	conversationIDPattern = regexp.MustCompile(`^[CDG][A-Z0-9]{8,}$`)
	relativePattern       = regexp.MustCompile(`^now-(\d+)([mhdw])$`)
)

type ConversationRef struct {
	WorkspaceID    string
	ConversationID string
}

func ParseConversation(input, expectedWorkspaceID string) (ConversationRef, error) {
	if conversationIDPattern.MatchString(input) {
		return ConversationRef{ConversationID: input}, nil
	}
	u, err := url.Parse(input)
	if err != nil || u.Scheme != "https" || u.Host != "app.slack.com" {
		return ConversationRef{}, errors.New("conversation must be a Slack URL or ID")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "client" || !workspaceIDPattern.MatchString(parts[1]) || !conversationIDPattern.MatchString(parts[2]) {
		return ConversationRef{}, errors.New("malformed Slack conversation URL")
	}
	if expectedWorkspaceID != "" && expectedWorkspaceID != parts[1] {
		return ConversationRef{}, errors.New("conversation workspace does not match credentials")
	}
	return ConversationRef{WorkspaceID: parts[1], ConversationID: parts[2]}, nil
}

type RequestInput struct {
	All  bool
	From string
	To   string
}

type TimeRange struct {
	From time.Time
	To   time.Time
	All  bool
}

func NormalizeRequest(in RequestInput, now time.Time) (TimeRange, error) {
	if !in.All && in.From == "" {
		return TimeRange{}, errors.New("either --all or --from is required")
	}
	if in.All && in.From != "" {
		return TimeRange{}, errors.New("--all and --from are mutually exclusive")
	}
	to := now.UTC()
	var err error
	if in.To != "" {
		to, err = parseTime(in.To, now)
		if err != nil {
			return TimeRange{}, fmt.Errorf("invalid --to: %w", err)
		}
	}
	var from time.Time
	if in.From != "" {
		from, err = parseTime(in.From, now)
		if err != nil {
			return TimeRange{}, fmt.Errorf("invalid --from: %w", err)
		}
	}
	if !from.IsZero() && from.After(to) {
		return TimeRange{}, errors.New("--from must not be after --to")
	}
	return TimeRange{From: from, To: to, All: in.All}, nil
}

func parseTime(value string, now time.Time) (time.Time, error) {
	if value == "now" {
		return now.UTC(), nil
	}
	if match := relativePattern.FindStringSubmatch(value); match != nil {
		n, _ := strconv.Atoi(match[1])
		unit := map[string]time.Duration{"m": time.Minute, "h": time.Hour, "d": 24 * time.Hour, "w": 7 * 24 * time.Hour}[match[2]]
		return now.UTC().Add(-time.Duration(n) * unit), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New("expected RFC3339, now, or now-<n>[m|h|d|w]")
	}
	return parsed.UTC(), nil
}
