package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

type RawMessage = slack.Message
type RawAttachment = slack.Attachment

type Participant struct {
	DisplayName string `json:"display_name"`
	RealName    string `json:"real_name,omitempty"`
	Deleted     bool   `json:"deleted,omitempty"`
	Unresolved  bool   `json:"unresolved,omitempty"`
}

type ConversationDTO struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type ReactionDTO struct {
	Name    string   `json:"name"`
	Count   int      `json:"count"`
	UserIDs []string `json:"user_ids"`
}

type ReactionList []ReactionDTO

func (reactions ReactionList) MarshalJSON() ([]byte, error) {
	if reactions == nil {
		return []byte("[]"), nil
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode([]ReactionDTO(reactions)); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(output.Bytes(), []byte("\n")), nil
}

type MessageDTO struct {
	Timestamp        string       `json:"timestamp"`
	Datetime         string       `json:"datetime"`
	AuthorID         string       `json:"author_id,omitempty"`
	AuthorName       string       `json:"author_name"`
	Text             string       `json:"text"`
	Reactions        ReactionList `json:"reactions"`
	ThreadReplyCount int          `json:"-"`
	ThreadReplies    []MessageDTO `json:"thread_replies"`
}

type Document struct {
	SchemaVersion int                    `json:"schema_version"`
	Conversation  ConversationDTO        `json:"conversation"`
	Participants  map[string]Participant `json:"participants"`
	Messages      []MessageDTO           `json:"messages"`
}

func Normalize(conversationID, conversationType string, roots []RawMessage, replies map[string][]RawMessage, participants map[string]Participant) Document {
	sortMessages(roots)
	messages := make([]MessageDTO, 0, len(roots))
	for _, root := range roots {
		item := normalizeMessage(root, participants)
		item.ThreadReplyCount = root.ReplyCount
		thread := append([]RawMessage(nil), replies[root.Timestamp]...)
		sortMessages(thread)
		item.ThreadReplies = make([]MessageDTO, 0, len(thread))
		for _, reply := range thread {
			item.ThreadReplies = append(item.ThreadReplies, normalizeMessage(reply, participants))
		}
		messages = append(messages, item)
	}
	if participants == nil {
		participants = map[string]Participant{}
	}
	return Document{SchemaVersion: 2, Conversation: ConversationDTO{ID: conversationID, Type: conversationType}, Participants: participants, Messages: messages}
}

func normalizeMessage(message RawMessage, participants map[string]Participant) MessageDTO {
	authorID, authorName := message.User, ""
	if message.User != "" {
		if participant, ok := participants[message.User]; ok && participant.DisplayName != "" {
			authorName = participant.DisplayName
			if participant.Unresolved {
				authorName += " (unresolved)"
			}
		} else {
			authorName = message.User + " (unresolved)"
		}
	} else if message.BotID != "" {
		authorID = message.BotID
		authorName = message.Username
		if authorName == "" {
			authorName = message.BotID + " (bot)"
		}
	} else {
		authorName = "unknown"
	}
	text := message.Text
	if text == "" {
		for _, attachment := range message.Attachments {
			if attachment.Text != "" {
				text = attachment.Text
				break
			}
			if attachment.Fallback != "" {
				text = attachment.Fallback
				break
			}
		}
	}
	return MessageDTO{
		Timestamp:     message.Timestamp,
		Datetime:      timestampTime(message.Timestamp).Format(time.RFC3339Nano),
		AuthorID:      authorID,
		AuthorName:    authorName,
		Text:          text,
		Reactions:     normalizeReactions(message.Reactions),
		ThreadReplies: []MessageDTO{},
	}
}

func normalizeReactions(raw []slack.Reaction) ReactionList {
	reactions := make(ReactionList, 0, len(raw))
	for _, reaction := range raw {
		seen := make(map[string]struct{}, len(reaction.Users))
		userIDs := make([]string, 0, len(reaction.Users))
		for _, userID := range reaction.Users {
			if _, exists := seen[userID]; exists {
				continue
			}
			seen[userID] = struct{}{}
			userIDs = append(userIDs, userID)
		}
		sort.Strings(userIDs)
		reactions = append(reactions, ReactionDTO{
			Name:    reaction.Name,
			Count:   reaction.Count,
			UserIDs: userIDs,
		})
	}
	sort.SliceStable(reactions, func(i, j int) bool {
		return reactions[i].Name < reactions[j].Name
	})
	return reactions
}

func MarshalDocument(document Document) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func sortMessages(messages []RawMessage) {
	sort.SliceStable(messages, func(i, j int) bool { return compareTimestamp(messages[i].Timestamp, messages[j].Timestamp) < 0 })
}

func compareTimestamp(left, right string) int {
	lf, _ := strconv.ParseFloat(left, 64)
	rf, _ := strconv.ParseFloat(right, 64)
	switch {
	case lf < rf:
		return -1
	case lf > rf:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func timestampTime(timestamp string) time.Time {
	parts := strings.SplitN(timestamp, ".", 2)
	seconds, _ := strconv.ParseInt(parts[0], 10, 64)
	var nanoseconds int64
	if len(parts) == 2 {
		fraction := (parts[1] + "000000000")[:9]
		nanoseconds, _ = strconv.ParseInt(fraction, 10, 64)
	}
	return time.Unix(seconds, nanoseconds).UTC()
}

var slackMarkup = regexp.MustCompile(`<([^>]+)>`)

func RenderMarkdown(document Document) string {
	var output strings.Builder
	fmt.Fprintf(&output, "# Slack conversation export — %s\n\n", document.Conversation.ID)
	for _, message := range document.Messages {
		fmt.Fprintf(&output, "**%s — %s**\n\n%s\n", markdownTime(message.Datetime), escapeMarkdown(message.AuthorName), renderSlackText(message.Text, document.Participants))
		if reactions := renderReactions(message.Reactions, document.Participants); reactions != "" {
			fmt.Fprintf(&output, "\n%s\n", reactions)
		}
		output.WriteString("\n")
		for _, reply := range message.ThreadReplies {
			fmt.Fprintf(&output, "> **%s — %s**\n>\n", markdownTime(reply.Datetime), escapeMarkdown(reply.AuthorName))
			for _, line := range strings.Split(renderSlackText(reply.Text, document.Participants), "\n") {
				fmt.Fprintf(&output, "> %s\n", line)
			}
			if reactions := renderReactions(reply.Reactions, document.Participants); reactions != "" {
				fmt.Fprintf(&output, "> %s\n", reactions)
			}
			output.WriteString("\n")
		}
	}
	return output.String()
}

func renderReactions(reactions []ReactionDTO, participants map[string]Participant) string {
	if len(reactions) == 0 {
		return ""
	}
	ordered := append([]ReactionDTO(nil), reactions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	rendered := make([]string, 0, len(ordered))
	for _, reaction := range ordered {
		value := ":" + escapeMarkdown(reaction.Name) + ": x" + strconv.Itoa(reaction.Count)
		userIDs := append([]string(nil), reaction.UserIDs...)
		sort.Strings(userIDs)
		names := make([]string, 0, len(userIDs))
		previous := ""
		for _, userID := range userIDs {
			if userID == previous {
				continue
			}
			previous = userID
			name := userID + " (unresolved)"
			if participant, exists := participants[userID]; exists && participant.DisplayName != "" {
				name = participant.DisplayName
				if participant.Unresolved {
					name += " (unresolved)"
				}
			}
			names = append(names, escapeMarkdown(name))
		}
		if len(names) > 0 {
			value += " - returned users: " + strings.Join(names, ", ")
		}
		rendered = append(rendered, value)
	}
	return "_Reactions: " + strings.Join(rendered, "; ") + "_"
}

func markdownTime(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return parsed.UTC().Format("2006-01-02 15:04:05 UTC")
}

func renderSlackText(text string, participants map[string]Participant) string {
	var output strings.Builder
	last := 0
	for _, index := range slackMarkup.FindAllStringSubmatchIndex(text, -1) {
		output.WriteString(escapeMarkdown(text[last:index[0]]))
		content := text[index[2]:index[3]]
		switch {
		case strings.HasPrefix(content, "@"):
			id := strings.TrimPrefix(content, "@")
			if participant, ok := participants[id]; ok && participant.DisplayName != "" {
				output.WriteString("@" + escapeMarkdown(participant.DisplayName))
				if participant.Unresolved {
					output.WriteString(" (unresolved)")
				}
			} else {
				output.WriteString("@" + id + " (unresolved)")
			}
		case strings.Contains(content, "|"):
			target, label, _ := strings.Cut(content, "|")
			fmt.Fprintf(&output, "[%s](%s)", escapeMarkdown(label), target)
		case strings.HasPrefix(content, "https://") || strings.HasPrefix(content, "http://"):
			output.WriteString(content)
		default:
			output.WriteString(escapeMarkdown(text[index[0]:index[1]]))
		}
		last = index[1]
	}
	output.WriteString(escapeMarkdown(text[last:]))
	return output.String()
}

func escapeMarkdown(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		"`", "\\`",
		`*`, `\*`,
		`_`, `\_`,
		`[`, `\[`,
		`]`, `\]`,
		`<`, `\<`,
		`>`, `\>`,
		`#`, `\#`,
		`~`, `\~`,
	)
	return replacer.Replace(value)
}
