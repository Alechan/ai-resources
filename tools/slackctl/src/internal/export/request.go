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
	permalinkTSPattern    = regexp.MustCompile(`^p[0-9]{16}$`)
	slackTSPattern        = regexp.MustCompile(`^[0-9]+\.[0-9]{6}$`)
	workspaceHostPattern  = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.slack\.com$`)
	relativePattern       = regexp.MustCompile(`^now-(\d+)([mhdw])$`)
)

type ConversationRefKind string

const (
	ConversationRefID        ConversationRefKind = "id"
	ConversationRefAppURL    ConversationRefKind = "app_url"
	ConversationRefPermalink ConversationRefKind = "permalink"
)

type ConversationRef struct {
	WorkspaceHost     string
	WorkspaceID       string
	ConversationID    string
	StartingTimestamp string
	Kind              ConversationRefKind
}

func ParseConversation(input, expectedWorkspaceID string) (ConversationRef, error) {
	if conversationIDPattern.MatchString(input) {
		return ConversationRef{ConversationID: input, Kind: ConversationRefID}, nil
	}
	u, err := url.Parse(input)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Host == "" || u.Host != u.Hostname() {
		return ConversationRef{}, errors.New("conversation must be a Slack URL or ID")
	}
	if u.RawPath != "" {
		return ConversationRef{}, errors.New("malformed Slack conversation URL")
	}

	switch u.Host {
	case "app.slack.com":
		return parseAppURL(u, expectedWorkspaceID)
	default:
		if !ValidWorkspaceHost(u.Host) {
			return ConversationRef{}, errors.New("conversation URL must use a Slack workspace host")
		}
		return parseArchivePermalink(u)
	}
}

func parseAppURL(u *url.URL, expectedWorkspaceID string) (ConversationRef, error) {
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "client" || !validWorkspaceID(parts[1]) || !validConversationID(parts[2]) {
		return ConversationRef{}, errors.New("malformed Slack conversation URL")
	}
	if expectedWorkspaceID != "" && expectedWorkspaceID != parts[1] {
		return ConversationRef{}, errors.New("conversation workspace does not match credentials")
	}
	return ConversationRef{WorkspaceID: parts[1], ConversationID: parts[2], Kind: ConversationRefAppURL}, nil
}

func parseArchivePermalink(u *url.URL) (ConversationRef, error) {
	if _, present := u.Query()["thread_ts"]; present {
		return ConversationRef{}, errors.New("reply permalinks are not supported; use the root-message permalink")
	}
	if !strings.HasPrefix(u.Path, "/") || strings.HasSuffix(u.Path, "/") || strings.Contains(u.Path, "//") {
		return ConversationRef{}, errors.New("malformed Slack archive permalink")
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "archives" || !validConversationID(parts[1]) {
		return ConversationRef{}, errors.New("malformed Slack archive permalink")
	}
	timestamp, err := parsePermalinkTimestamp(parts[2])
	if err != nil {
		return ConversationRef{}, err
	}
	return ConversationRef{
		WorkspaceHost:     u.Host,
		ConversationID:    parts[1],
		StartingTimestamp: timestamp,
		Kind:              ConversationRefPermalink,
	}, nil
}

func parsePermalinkTimestamp(value string) (string, error) {
	if !permalinkTSPattern.MatchString(value) {
		return "", errors.New("malformed Slack permalink timestamp")
	}
	digits := strings.TrimPrefix(value, "p")
	return digits[:len(digits)-6] + "." + digits[len(digits)-6:], nil
}

func validWorkspaceID(value string) bool    { return workspaceIDPattern.MatchString(value) }
func validConversationID(value string) bool { return conversationIDPattern.MatchString(value) }

func ValidWorkspaceHost(host string) bool {
	return host != "slack.com" && workspaceHostPattern.MatchString(host)
}

type RequestInput struct {
	All           bool
	From          string
	To            string
	PermalinkFrom string
}

type RangeFromSource string

const (
	RangeFromNone      RangeFromSource = ""
	RangeFromExplicit  RangeFromSource = "explicit"
	RangeFromPermalink RangeFromSource = "permalink"
)

type TimeRange struct {
	From       time.Time
	To         time.Time
	All        bool
	FromSource RangeFromSource
}

func NormalizeRequest(in RequestInput, now time.Time) (TimeRange, error) {
	if in.PermalinkFrom != "" && in.All {
		return TimeRange{}, errors.New("--all cannot be used with a message permalink")
	}
	if in.PermalinkFrom != "" && in.From != "" {
		return TimeRange{}, errors.New("--from cannot be used with a message permalink")
	}
	if !in.All && in.From == "" && in.PermalinkFrom == "" {
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
	fromSource := RangeFromNone
	if in.From != "" {
		from, err = parseTime(in.From, now)
		if err != nil {
			return TimeRange{}, fmt.Errorf("invalid --from: %w", err)
		}
		fromSource = RangeFromExplicit
	} else if in.PermalinkFrom != "" {
		from, err = parseSlackTimestamp(in.PermalinkFrom)
		if err != nil {
			return TimeRange{}, fmt.Errorf("invalid permalink timestamp: %w", err)
		}
		fromSource = RangeFromPermalink
	}
	if !from.IsZero() && from.After(to) {
		if fromSource == RangeFromPermalink {
			return TimeRange{}, errors.New("permalink lower boundary must not be after --to")
		}
		return TimeRange{}, errors.New("--from must not be after --to")
	}
	return TimeRange{From: from, To: to, All: in.All, FromSource: fromSource}, nil
}

func parseSlackTimestamp(value string) (time.Time, error) {
	if !slackTSPattern.MatchString(value) {
		return time.Time{}, errors.New("expected Slack seconds.microseconds timestamp")
	}
	secondsValue, fractionValue, _ := strings.Cut(value, ".")
	seconds, err := strconv.ParseInt(secondsValue, 10, 64)
	if err != nil {
		return time.Time{}, errors.New("Slack timestamp seconds are out of range")
	}
	microseconds, _ := strconv.ParseInt(fractionValue, 10, 64)
	return time.Unix(seconds, microseconds*int64(time.Microsecond)).UTC(), nil
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
