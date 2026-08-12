package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

type API interface {
	History(context.Context, string, string, string, string, int) (slack.HistoryPage, []byte, error)
	Replies(context.Context, string, string, string, int) (slack.HistoryPage, []byte, error)
	UserInfo(context.Context, string) (slack.User, error)
	ConversationInfo(context.Context, string) (slack.Conversation, error)
}

type Format string

const (
	FormatRaw      Format = "raw"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

type Options struct {
	WorkspaceHost  string
	WorkspaceID    string
	ConversationID string
	Range          TimeRange
	IncludeThreads bool
	ResolveUsers   bool
	PageSize       int
	OutputDir      string
	Formats        map[Format]bool
	Resume         bool
	AllowPartial   bool
	Secrets        []string
	Progress       func(string)
	InMemory       bool
}

type Manifest struct {
	SchemaVersion    int     `json:"schema_version"`
	WorkspaceHost    string  `json:"workspace_host"`
	WorkspaceID      string  `json:"workspace_id"`
	ConversationID   string  `json:"conversation_id"`
	ConversationType string  `json:"conversation_type"`
	ExportedAt       string  `json:"exported_at"`
	RequestedFrom    *string `json:"requested_from"`
	RequestedTo      string  `json:"requested_to"`
	OldestExported   string  `json:"oldest_exported,omitempty"`
	NewestExported   string  `json:"newest_exported,omitempty"`
	RootMessageCount int     `json:"root_message_count"`
	ThreadReplyCount int     `json:"thread_reply_count"`
	ParticipantCount int     `json:"participant_count"`
	Complete         bool    `json:"complete"`
}

type Result struct {
	OutputDir    string
	Manifest     Manifest
	Document     Document
	Warnings     []string
	RawResponses []RawResponse
}

type RawResponse struct {
	Kind     string          `json:"kind"`
	ThreadTS string          `json:"thread_ts,omitempty"`
	Page     int             `json:"page"`
	Response json.RawMessage `json:"response"`
}

type Exporter struct {
	api          API
	now          func() time.Time
	rawResponses []RawResponse
}

func NewExporter(api API, now func() time.Time) *Exporter { return &Exporter{api: api, now: now} }

func (e *Exporter) Run(ctx context.Context, options Options) (Result, error) {
	if options.PageSize < 1 || options.PageSize > 1000 {
		return Result{}, errors.New("page size must be between 1 and 1000")
	}
	if options.OutputDir == "" && !options.InMemory {
		return Result{}, errors.New("output directory is required")
	}
	if options.Progress == nil {
		options.Progress = func(string) {}
	}
	e.rawResponses = nil
	if !options.InMemory {
		if err := prepareOutput(options); err != nil {
			return Result{}, err
		}
	}
	manifest := newManifest(options, e.now())
	if !options.InMemory {
		if options.Resume {
			if err := validateResumeManifest(filepath.Join(options.OutputDir, "manifest.json"), manifest); err != nil {
				return Result{}, err
			}
		}
		if err := writeJSONAtomic(filepath.Join(options.OutputDir, "manifest.json"), manifest); err != nil {
			return Result{}, err
		}

	}
	fail := func(err error) (Result, error) {
		if !options.InMemory {
			_ = writeJSONAtomic(filepath.Join(options.OutputDir, "manifest.json"), manifest)
		}
		return Result{OutputDir: options.OutputDir, Manifest: manifest, RawResponses: e.rawResponses}, err
	}

	conversation, err := e.api.ConversationInfo(ctx, options.ConversationID)
	if err != nil {
		return fail(fmt.Errorf("conversation lookup: %w", err))
	}
	manifest.ConversationType = conversationType(conversation)
	var warnings []string
	complete := true
	roots, err := e.fetchHistory(ctx, options)
	if err != nil {
		if !options.AllowPartial || len(roots) == 0 {
			return fail(err)
		}
		warnings = append(warnings, err.Error())
		complete = false
	}
	replies := make(map[string][]slack.Message)
	fetchedThreads := make(map[string]bool)
	if options.IncludeThreads {
		for _, root := range roots {
			if root.ReplyCount < 1 {
				continue
			}
			thread, threadErr := e.fetchThread(ctx, options, root)
			if threadErr != nil {
				if !options.AllowPartial {
					return fail(threadErr)
				}
				warnings = append(warnings, threadErr.Error())
				complete = false
				continue
			}
			replies[root.Timestamp] = thread
			fetchedThreads[root.Timestamp] = true
		}
	} else {
		for _, root := range roots {
			if root.ReplyCount > 0 {
				fetchedThreads[root.Timestamp] = true
			}
		}
	}
	participants, resolutionWarnings := e.resolveParticipants(ctx, options, roots, replies)
	warnings = append(warnings, resolutionWarnings...)
	document := Normalize(options.ConversationID, manifest.ConversationType, roots, replies, participants)
	populateManifest(&manifest, document, complete)
	if err := Validate(manifest, document, fetchedThreads); err != nil {
		if !options.AllowPartial {
			return fail(err)
		}
		manifest.Complete = false
		warnings = append(warnings, err.Error())
	}
	if options.InMemory {
		if err := scanMemorySecrets(document, e.rawResponses, options.Secrets); err != nil {
			return fail(err)
		}
	} else {
		if err := writeSelectedOutputs(options, document); err != nil {
			return fail(err)
		}
		if paths, err := ScanSecrets(options.OutputDir, options.Secrets); err != nil {
			for _, path := range paths {
				_ = os.Remove(path)
			}
			return fail(err)
		}
		if !options.Formats[FormatRaw] {
			_ = os.RemoveAll(filepath.Join(options.OutputDir, "raw_history"))
			_ = os.RemoveAll(filepath.Join(options.OutputDir, "raw_threads"))
		}
		if err := writeJSONAtomic(filepath.Join(options.OutputDir, "manifest.json"), manifest); err != nil {
			return fail(err)
		}
	}
	return Result{OutputDir: options.OutputDir, Manifest: manifest, Document: document, Warnings: warnings, RawResponses: e.rawResponses}, nil
}

func validateResumeManifest(path string, requested Manifest) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.New("resume request requires an existing manifest")
	}
	var existing Manifest
	if err := json.Unmarshal(data, &existing); err != nil {
		return errors.New("resume request has an invalid existing manifest")
	}
	if existing.SchemaVersion != requested.SchemaVersion ||
		existing.WorkspaceHost != requested.WorkspaceHost ||
		existing.WorkspaceID != requested.WorkspaceID ||
		existing.ConversationID != requested.ConversationID ||
		existing.RequestedTo != requested.RequestedTo ||
		!equalOptionalString(existing.RequestedFrom, requested.RequestedFrom) {
		return errors.New("resume request does not match the existing export")
	}
	return nil
}

func equalOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func prepareOutput(options Options) error {
	if !options.Resume {
		if entries, err := os.ReadDir(options.OutputDir); err == nil && len(entries) > 0 {
			return errors.New("output directory is not empty; use --resume or choose another directory")
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, directory := range []string{options.OutputDir, filepath.Join(options.OutputDir, "raw_history"), filepath.Join(options.OutputDir, "raw_threads")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return fmt.Errorf("secure output directory: %w", err)
		}
	}
	return nil
}

func newManifest(options Options, now time.Time) Manifest {
	manifest := Manifest{
		SchemaVersion:  1,
		WorkspaceHost:  options.WorkspaceHost,
		WorkspaceID:    options.WorkspaceID,
		ConversationID: options.ConversationID,
		ExportedAt:     now.UTC().Format(time.RFC3339),
		RequestedTo:    options.Range.To.UTC().Format(time.RFC3339Nano),
	}
	if !options.Range.From.IsZero() {
		value := options.Range.From.UTC().Format(time.RFC3339Nano)
		manifest.RequestedFrom = &value
	}
	return manifest
}

func (e *Exporter) fetchHistory(ctx context.Context, options Options) ([]slack.Message, error) {
	var pages []slack.HistoryPage
	directory := ""
	if !options.InMemory {
		directory = filepath.Join(options.OutputDir, "raw_history")
		var err error
		pages, err = loadPages(directory, "page_*.json")
		if err != nil {
			return nil, err
		}
	}
	if len(pages) > 0 && !options.Resume {
		return nil, errors.New("raw history already exists")
	}
	messages := make(map[string]slack.Message)
	latest := formatSlackTimestamp(options.Range.To)
	hasMore := true
	for _, page := range pages {
		if err := validateMessages(page.Messages); err != nil {
			return nil, err
		}
		addMessages(messages, page.Messages, options.Range)
		if len(page.Messages) > 0 {
			latest = oldestTimestamp(page.Messages)
		}
		hasMore = page.HasMore
	}
	pageNumber := len(pages) + 1
	for hasMore {
		options.Progress(fmt.Sprintf("history page %d", pageNumber))
		page, raw, callErr := e.api.History(ctx, options.ConversationID, formatSlackTimestamp(options.Range.From), latest, "", options.PageSize)
		if callErr != nil {
			return sortedMapMessages(messages), fmt.Errorf("history page %d: %w", pageNumber, callErr)
		}
		if options.InMemory {
			e.rawResponses = append(e.rawResponses, RawResponse{Kind: "history", Page: pageNumber, Response: append(json.RawMessage(nil), raw...)})
		} else if err := writeRawPage(filepath.Join(directory, fmt.Sprintf("page_%04d.json", pageNumber)), raw, options.Secrets); err != nil {
			return nil, err
		}
		if err := validateMessages(page.Messages); err != nil {
			return nil, err
		}
		addMessages(messages, page.Messages, options.Range)
		if !page.HasMore || len(page.Messages) == 0 {
			break
		}
		next := oldestTimestamp(page.Messages)
		if compareTimestamp(next, latest) >= 0 {
			return sortedMapMessages(messages), errors.New("history pagination did not advance")
		}
		latest = next
		pageNumber++
	}
	return sortedMapMessages(messages), nil
}

func (e *Exporter) fetchThread(ctx context.Context, options Options, root slack.Message) ([]slack.Message, error) {
	directory := ""
	prefix := "thread_" + sanitizeTimestamp(root.Timestamp)
	var pages []slack.HistoryPage
	if !options.InMemory {
		directory = filepath.Join(options.OutputDir, "raw_threads")
		var err error
		pages, err = loadPages(directory, prefix+"_page_*.json")
		if err != nil {
			return nil, err
		}
	}
	replies := make(map[string]slack.Message)
	cursor := ""
	hasMore := false
	for _, page := range pages {
		if err := validateMessages(page.Messages); err != nil {
			return nil, err
		}
		if err := validateThreadParents(page.Messages, root.Timestamp); err != nil {
			return nil, err
		}
		addThreadMessages(replies, page.Messages, root.Timestamp)
		cursor = page.ResponseMetadata.NextCursor
		hasMore = page.HasMore
	}
	if len(pages) > 0 && hasMore && cursor == "" {
		return nil, fmt.Errorf("thread %s response is missing a pagination cursor", root.Timestamp)
	}
	pageNumber := len(pages) + 1
	if len(pages) == 0 || cursor != "" {
		for {
			options.Progress(fmt.Sprintf("thread %s page %d", root.Timestamp, pageNumber))
			page, raw, callErr := e.api.Replies(ctx, options.ConversationID, root.Timestamp, cursor, options.PageSize)
			if callErr != nil {
				return nil, fmt.Errorf("thread %s page %d: %w", root.Timestamp, pageNumber, callErr)
			}
			if options.InMemory {
				e.rawResponses = append(e.rawResponses, RawResponse{Kind: "thread", ThreadTS: root.Timestamp, Page: pageNumber, Response: append(json.RawMessage(nil), raw...)})
			} else if err := writeRawPage(filepath.Join(directory, fmt.Sprintf("%s_page_%04d.json", prefix, pageNumber)), raw, options.Secrets); err != nil {
				return nil, err
			}
			if err := validateMessages(page.Messages); err != nil {
				return nil, err
			}
			if err := validateThreadParents(page.Messages, root.Timestamp); err != nil {
				return nil, err
			}
			addThreadMessages(replies, page.Messages, root.Timestamp)
			next := page.ResponseMetadata.NextCursor
			if page.HasMore && next == "" {
				return nil, fmt.Errorf("thread %s response is missing a pagination cursor", root.Timestamp)
			}
			if next == "" {
				break
			}
			if next == cursor {
				return nil, fmt.Errorf("thread %s pagination did not advance", root.Timestamp)
			}
			cursor = next
			pageNumber++
		}
	}
	result := mapMessages(replies)
	sortMessages(result)
	return result, nil
}

func (e *Exporter) resolveParticipants(ctx context.Context, options Options, roots []slack.Message, replies map[string][]slack.Message) (map[string]Participant, []string) {
	ids := make(map[string]bool)
	for _, message := range roots {
		if message.User != "" {
			ids[message.User] = true
		} else if message.BotID != "" {
			ids[message.BotID] = true
		}
		for _, reply := range replies[message.Timestamp] {
			if reply.User != "" {
				ids[reply.User] = true
			} else if reply.BotID != "" {
				ids[reply.BotID] = true
			}
		}
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	participants := make(map[string]Participant, len(ordered))
	var warnings []string
	for _, id := range ordered {
		if strings.HasPrefix(id, "B") {
			name := botName(id, roots, replies)
			participants[id] = Participant{DisplayName: name}
			continue
		}
		if !options.ResolveUsers {
			participants[id] = Participant{DisplayName: id, Unresolved: true}
			continue
		}

		user, err := e.api.UserInfo(ctx, id)
		if err != nil {
			participants[id] = Participant{DisplayName: id, Unresolved: true}
			warnings = append(warnings, "could not resolve participant "+id)
			continue
		}
		name := user.Profile.DisplayName
		if name == "" {
			name = user.Profile.RealName
		}
		if name == "" {
			name = user.Name
		}
		if name == "" {
			name = id
		}
		participants[id] = Participant{DisplayName: name, RealName: user.Profile.RealName, Deleted: user.Deleted, Unresolved: name == id}
	}
	return participants, warnings
}

func botName(id string, roots []slack.Message, replies map[string][]slack.Message) string {
	for _, root := range roots {
		if root.BotID == id && root.Username != "" {
			return root.Username
		}
		for _, reply := range replies[root.Timestamp] {
			if reply.BotID == id && reply.Username != "" {
				return reply.Username
			}
		}
	}
	return id + " (bot)"
}

func loadPages(directory, pattern string) ([]slack.HistoryPage, error) {
	paths, err := filepath.Glob(filepath.Join(directory, pattern))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	pages := make([]slack.HistoryPage, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var page slack.HistoryPage
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, fmt.Errorf("invalid raw page %s", filepath.Base(path))
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func writeRawPage(path string, data []byte, secrets []string) error {
	if _, err := os.Lstat(path); err == nil {
		return errors.New("refusing to overwrite an existing raw response")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var status struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(data, &status); err != nil || !status.OK {
		return errors.New("refusing to persist malformed raw response")
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(string(data), secret) {
			return errors.New("credential leak detected in raw response; page was not persisted")
		}
	}
	return writeAtomic(path, append(data, '\n'), 0o600)
}

func writeSelectedOutputs(options Options, document Document) error {
	if options.Formats[FormatJSON] {
		data, err := MarshalDocument(document)
		if err != nil {
			return err
		}
		if err := writeAtomic(filepath.Join(options.OutputDir, "messages_with_threads.json"), data, 0o600); err != nil {
			return err
		}
	}
	if options.Formats[FormatMarkdown] {
		if err := writeAtomic(filepath.Join(options.OutputDir, "conversation.md"), []byte(RenderMarkdown(document)), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(data, '\n'), 0o600)
}

func writeAtomic(path string, data []byte, mode fs.FileMode) error {
	partial := path + ".partial"
	defer os.Remove(partial)
	if err := os.Remove(partial); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(partial, path)
}

func addMessages(destination map[string]slack.Message, source []slack.Message, requested TimeRange) {
	for _, message := range source {
		timestamp := timestampTime(message.Timestamp)
		if (!requested.From.IsZero() && timestamp.Before(requested.From)) || timestamp.After(requested.To) {
			continue
		}

		destination[message.Timestamp] = message
	}
}

func validateMessages(messages []slack.Message) error {
	for _, message := range messages {
		parts := strings.Split(message.Timestamp, ".")
		if len(parts) != 2 || parts[0] == "" || len(parts[1]) < 1 || len(parts[1]) > 9 ||
			!allDigits(parts[0]) || !allDigits(parts[1]) {
			return errors.New("Slack response contains an invalid message timestamp")
		}
	}
	return nil
}

func validateThreadParents(messages []slack.Message, rootTimestamp string) error {
	for _, message := range messages {
		if message.Timestamp != rootTimestamp && message.ThreadTS != "" && message.ThreadTS != rootTimestamp {
			return errors.New("thread reply parent does not match the selected root")
		}
	}
	return nil
}

func allDigits(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func addThreadMessages(destination map[string]slack.Message, source []slack.Message, rootTimestamp string) {
	for _, message := range source {
		if message.Timestamp == rootTimestamp {
			continue
		}
		destination[message.Timestamp] = message
	}
}

func mapMessages(source map[string]slack.Message) []slack.Message {
	result := make([]slack.Message, 0, len(source))
	for _, message := range source {
		result = append(result, message)
	}

	return result
}

func sortedMapMessages(source map[string]slack.Message) []slack.Message {
	result := mapMessages(source)
	sortMessages(result)
	return result
}

func oldestTimestamp(messages []slack.Message) string {
	oldest := messages[0].Timestamp
	for _, message := range messages[1:] {
		if compareTimestamp(message.Timestamp, oldest) < 0 {
			oldest = message.Timestamp
		}
	}
	return oldest
}

func formatSlackTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d.%06d", value.Unix(), value.Nanosecond()/1000)
}

func sanitizeTimestamp(timestamp string) string { return strings.ReplaceAll(timestamp, ".", "_") }

func conversationType(conversation slack.Conversation) string {
	switch {
	case conversation.IsIM:
		return "direct_message"
	case conversation.IsMPIM:
		return "group_direct_message"
	case conversation.IsPrivate || conversation.IsGroup:
		return "private_channel"
	case conversation.IsChannel:
		return "channel"
	default:
		return "unknown"
	}
}

func populateManifest(manifest *Manifest, document Document, complete bool) {
	manifest.RootMessageCount = len(document.Messages)
	manifest.ParticipantCount = len(document.Participants)
	manifest.Complete = complete
	var timestamps []string
	for _, message := range document.Messages {
		timestamps = append(timestamps, message.Timestamp)
		manifest.ThreadReplyCount += len(message.ThreadReplies)
		for _, reply := range message.ThreadReplies {
			timestamps = append(timestamps, reply.Timestamp)
		}
	}
	if len(timestamps) > 0 {
		sort.Slice(timestamps, func(i, j int) bool { return compareTimestamp(timestamps[i], timestamps[j]) < 0 })
		manifest.OldestExported = timestampTime(timestamps[0]).Format(time.RFC3339Nano)
		manifest.NewestExported = timestampTime(timestamps[len(timestamps)-1]).Format(time.RFC3339Nano)
	}
}

func Validate(manifest Manifest, document Document, fetchedThreads map[string]bool) error {
	if manifest.RootMessageCount != len(document.Messages) {
		return errors.New("manifest root message count does not match normalized output")
	}
	if manifest.ParticipantCount != len(document.Participants) {
		return errors.New("manifest participant count does not match normalized output")
	}
	var from time.Time
	if manifest.RequestedFrom != nil {
		from, _ = time.Parse(time.RFC3339, *manifest.RequestedFrom)
	}
	to, _ := time.Parse(time.RFC3339, manifest.RequestedTo)
	var previous string
	replyCount := 0
	for _, message := range document.Messages {
		if previous != "" && compareTimestamp(previous, message.Timestamp) > 0 {
			return errors.New("normalized messages are not chronological")
		}
		previous = message.Timestamp
		at := timestampTime(message.Timestamp)
		if (!from.IsZero() && at.Before(from)) || (!to.IsZero() && at.After(to)) {
			return errors.New("normalized message is outside requested range")
		}
		if message.ThreadReplyCount > 0 && !fetchedThreads[message.Timestamp] {
			return fmt.Errorf("thread %s was not fetched", message.Timestamp)
		}
		var previousReply string
		for _, reply := range message.ThreadReplies {
			replyCount++
			if previousReply != "" && compareTimestamp(previousReply, reply.Timestamp) > 0 {
				return errors.New("thread replies are not chronological")
			}
			previousReply = reply.Timestamp
		}
	}
	if manifest.ThreadReplyCount != replyCount {
		return errors.New("manifest thread reply count does not match normalized output")
	}
	return nil
}

func ScanSecrets(root string, secrets []string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("refusing to scan an export containing symbolic links")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, secret := range secrets {
			if secret != "" && strings.Contains(string(data), secret) {
				found = append(found, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		return found, err
	}
	if len(found) > 0 {
		return found, errors.New("credential leak detected in export; affected files removed")
	}
	return nil, nil
}

func scanMemorySecrets(document Document, raw []RawResponse, secrets []string) error {
	normalized, _ := MarshalDocument(document)
	markdown := []byte(RenderMarkdown(document))
	rawData, _ := json.Marshal(raw)
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		for _, data := range [][]byte{normalized, markdown, rawData} {
			if strings.Contains(string(data), secret) {
				return errors.New("credential leak detected in export")
			}
		}
	}
	return nil
}
