package export

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

type fakeAPI struct {
	history      []slack.HistoryPage
	historyCalls int
	replies      map[string][]slack.HistoryPage
	replyCalls   map[string]int
	users        map[string]slack.User
	userCalls    map[string]int
	userErrors   map[string]error
	conversation slack.Conversation
	historyErrAt int
	threadErrFor string
	threadErrAt  map[string]int
}

func (f *fakeAPI) History(_ context.Context, _, _, _, _ string, _ int) (slack.HistoryPage, []byte, error) {
	if f.historyCalls == f.historyErrAt && f.historyErrAt >= 0 {
		f.historyCalls++
		return slack.HistoryPage{}, nil, errors.New("synthetic interruption")
	}
	if f.historyCalls >= len(f.history) {
		return slack.HistoryPage{}, envelopeJSON(slack.HistoryPage{}), nil
	}
	page := f.history[f.historyCalls]
	f.historyCalls++
	return page, envelopeJSON(page), nil
}

func (f *fakeAPI) Replies(_ context.Context, _, timestamp, _ string, _ int) (slack.HistoryPage, []byte, error) {
	if timestamp == f.threadErrFor {
		return slack.HistoryPage{}, nil, errors.New("synthetic thread failure")
	}
	if f.replyCalls == nil {
		f.replyCalls = make(map[string]int)
	}
	index := f.replyCalls[timestamp]
	if errorIndex, ok := f.threadErrAt[timestamp]; ok && index == errorIndex {
		return slack.HistoryPage{}, nil, errors.New("synthetic thread interruption")
	}
	f.replyCalls[timestamp]++
	page := f.replies[timestamp][index]
	return page, envelopeJSON(page), nil
}

func (f *fakeAPI) UserInfo(_ context.Context, id string) (slack.User, error) {
	if f.userCalls == nil {
		f.userCalls = make(map[string]int)
	}
	f.userCalls[id]++
	if err := f.userErrors[id]; err != nil {
		return slack.User{}, err
	}
	return f.users[id], nil
}

func (f *fakeAPI) ConversationInfo(context.Context, string) (slack.Conversation, error) {
	return f.conversation, nil
}

func envelopeJSON(page slack.HistoryPage) []byte {
	data, _ := json.Marshal(struct {
		OK bool `json:"ok"`
		slack.HistoryPage
	}{true, page})
	return data
}

func testOptions(t *testing.T) Options {
	t.Helper()
	return Options{
		WorkspaceHost:  "alpha.slack.com",
		WorkspaceID:    "T11111111",
		ConversationID: "C22222222",
		Range:          TimeRange{All: true, To: time.Unix(200, 0).UTC()},
		IncludeThreads: true,
		ResolveUsers:   true,
		PageSize:       100,
		OutputDir:      t.TempDir(),
		Formats:        map[Format]bool{FormatRaw: true, FormatJSON: true, FormatMarkdown: true},
	}
}

func TestExporterPaginatesDeduplicatesThreadsAndUsers(t *testing.T) {
	root := slack.Message{Timestamp: "100.000001", User: "U11111111", Text: "root", ReplyCount: 2}
	api := &fakeAPI{
		historyErrAt: -1,
		history: []slack.HistoryPage{
			{Messages: []slack.Message{{Timestamp: "110.000001", User: "U22222222", Text: "new"}, root}, HasMore: true},
			{Messages: []slack.Message{root, {Timestamp: "90.000001", User: "U11111111", Text: "old"}}},
		},
		replies: map[string][]slack.HistoryPage{
			root.Timestamp: {
				{Messages: []slack.Message{root, {Timestamp: "101.000001", User: "U22222222", Text: "reply"}}, ResponseMetadata: slack.ResponseMetadata{NextCursor: "next"}},
				{Messages: []slack.Message{{Timestamp: "101.000001", User: "U22222222", Text: "reply"}, {Timestamp: "102.000001", User: "U11111111", Text: "last"}}},
			},
		},
		users: map[string]slack.User{
			"U11111111": {ID: "U11111111", Profile: slack.UserProfile{DisplayName: "Example One", RealName: "Example Person One"}},
			"U22222222": {ID: "U22222222", Profile: slack.UserProfile{RealName: "Example Person Two"}},
		},
		conversation: slack.Conversation{ID: "C22222222", IsChannel: true, IsPrivate: true},
	}
	result, err := NewExporter(api, func() time.Time { return time.Unix(201, 0).UTC() }).Run(t.Context(), testOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.RootMessageCount != 3 || result.Manifest.ThreadReplyCount != 2 || result.Manifest.ParticipantCount != 2 || !result.Manifest.Complete {
		t.Fatalf("manifest = %#v", result.Manifest)
	}
	if len(result.Document.Messages) != 3 || result.Document.Messages[1].Timestamp != root.Timestamp || len(result.Document.Messages[1].ThreadReplies) != 2 {
		t.Fatalf("messages = %#v", result.Document.Messages)
	}
	if result.Document.Participants["U11111111"].DisplayName != "Example One" || result.Document.Participants["U22222222"].DisplayName != "Example Person Two" {
		t.Fatalf("participants = %#v", result.Document.Participants)
	}
	if api.userCalls["U11111111"] != 1 || api.userCalls["U22222222"] != 1 {
		t.Fatalf("user calls = %#v", api.userCalls)
	}
	for _, path := range []string{"manifest.json", "messages_with_threads.json", "conversation.md", "raw_history/page_0001.json", "raw_threads/thread_100_000001_page_0001.json"} {
		if _, err := os.Stat(filepath.Join(testOptionsPath(result, api, path))); err != nil {
			t.Errorf("expected %s: %v", path, err)
		}
	}
}

func TestExporterIncludesPermalinkBoundaryRootExactlyOnce(t *testing.T) {
	options := testOptions(t)
	options.Range = TimeRange{
		From:       time.Unix(1786455295, 71869000).UTC(),
		To:         time.Unix(1786456000, 0).UTC(),
		FromSource: RangeFromPermalink,
	}
	boundary := slack.Message{Timestamp: "1786455295.071869", Text: "boundary root"}
	api := &fakeAPI{
		historyErrAt: -1,
		history: []slack.HistoryPage{
			{
				Messages: []slack.Message{
					{Timestamp: "1786455296.000001", Text: "later root"},
					boundary,
				},
				HasMore: true,
			},
			{Messages: []slack.Message{
				boundary,
				{Timestamp: "1786455295.071868", Text: "earlier root"},
			}},
		},
		replies: map[string][]slack.HistoryPage{},
	}
	result, err := NewExporter(api, time.Now).Run(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Document.Messages) != 2 ||
		result.Document.Messages[0].Timestamp != boundary.Timestamp ||
		result.Manifest.RootMessageCount != 2 {
		t.Fatalf("selected roots = %#v", result.Document.Messages)
	}
}

func testOptionsPath(result Result, _ *fakeAPI, path string) string {
	return filepath.Join(result.OutputDir, path)
}

func TestExporterRejectsNonAdvancingHistory(t *testing.T) {
	page := slack.HistoryPage{Messages: []slack.Message{{Timestamp: "100.000001"}}, HasMore: true}
	api := &fakeAPI{historyErrAt: -1, history: []slack.HistoryPage{page, page}, replies: map[string][]slack.HistoryPage{}}
	_, err := NewExporter(api, time.Now).Run(t.Context(), testOptions(t))
	if err == nil || !strings.Contains(err.Error(), "did not advance") {
		t.Fatalf("error = %v", err)
	}
}

func TestExporterResumeUsesPersistedHistory(t *testing.T) {
	options := testOptions(t)
	api := &fakeAPI{historyErrAt: 1, history: []slack.HistoryPage{
		{Messages: []slack.Message{{Timestamp: "100.000001", Text: "first"}}, HasMore: true},
		{Messages: []slack.Message{{Timestamp: "90.000001", Text: "second"}}},
	}, replies: map[string][]slack.HistoryPage{}}
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil {
		t.Fatal("first run should be interrupted")
	}

	var incomplete Manifest
	readJSONFile(t, filepath.Join(options.OutputDir, "manifest.json"), &incomplete)
	if incomplete.Complete {
		t.Fatal("interrupted manifest should be incomplete")
	}
	api.historyErrAt = -1
	api.historyCalls = 1
	options.Resume = true
	result, err := NewExporter(api, time.Now).Run(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.RootMessageCount != 2 || api.historyCalls != 2 {
		t.Fatalf("manifest=%#v calls=%d", result.Manifest, api.historyCalls)
	}
}

func TestExporterResumeRejectsDifferentRequest(t *testing.T) {
	options := testOptions(t)
	api := &fakeAPI{
		historyErrAt: 0,
		history:      []slack.HistoryPage{{Messages: []slack.Message{{Timestamp: "100.000001"}}}},
		replies:      map[string][]slack.HistoryPage{},
	}
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil {
		t.Fatal("first run should be interrupted")
	}
	options.Resume = true
	options.ConversationID = "C33333333"
	api.historyErrAt = -1
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil || !strings.Contains(err.Error(), "resume request") {
		t.Fatalf("error = %v", err)
	}
}

func TestExporterResumeCompletesPermalinkThreadPages(t *testing.T) {
	options := testOptions(t)
	options.Range = TimeRange{
		From:       time.Unix(100, 1_000).UTC(),
		To:         time.Unix(120, 0).UTC(),
		FromSource: RangeFromPermalink,
	}
	root := slack.Message{Timestamp: "100.000001", ReplyCount: 2}
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{root}}},
		replies: map[string][]slack.HistoryPage{
			root.Timestamp: {
				{
					Messages:         []slack.Message{root, {Timestamp: "110.000001", Text: "first reply"}},
					HasMore:          true,
					ResponseMetadata: slack.ResponseMetadata{NextCursor: "next"},
				},
				{Messages: []slack.Message{{Timestamp: "130.000001", Text: "late reply"}}},
			},
		},
		threadErrAt: map[string]int{root.Timestamp: 1},
	}
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil {
		t.Fatal("first run should be interrupted in the second thread page")
	}

	delete(api.threadErrAt, root.Timestamp)
	options.Resume = true
	result, err := NewExporter(api, time.Now).Run(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.ThreadReplyCount != 2 ||
		result.Manifest.NewestExported != "1970-01-01T00:02:10.000001Z" ||
		len(result.Document.Messages[0].ThreadReplies) != 2 {
		t.Fatalf("resumed result = %#v", result)
	}
	for _, path := range []string{
		"raw_threads/thread_100_000001_page_0001.json",
		"raw_threads/thread_100_000001_page_0002.json",
	} {
		if _, err := os.Stat(filepath.Join(options.OutputDir, path)); err != nil {
			t.Fatalf("missing deterministic raw page %s: %v", path, err)
		}
	}
}

func TestExporterResumeRejectsDifferentPermalinkBoundary(t *testing.T) {
	options := testOptions(t)
	options.Range = TimeRange{
		From:       time.Unix(100, 1_000).UTC(),
		To:         time.Unix(120, 0).UTC(),
		FromSource: RangeFromPermalink,
	}
	api := &fakeAPI{
		historyErrAt: 0,
		history:      []slack.HistoryPage{{Messages: []slack.Message{{Timestamp: "100.000001"}}}},
		replies:      map[string][]slack.HistoryPage{},
	}
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil {
		t.Fatal("first run should be interrupted")
	}
	options.Resume = true
	options.Range.From = time.Unix(100, 2_000).UTC()
	api.historyErrAt = -1
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("resume mismatch error = %v", err)
	}
}

func readJSONFile(t *testing.T, path string, output any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, output); err != nil {
		t.Fatal(err)
	}
}

func TestExporterContinuesWhenUserResolutionFails(t *testing.T) {
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{{Timestamp: "100.000001", User: "U11111111"}}}},
		replies:      map[string][]slack.HistoryPage{},
		userErrors:   map[string]error{"U11111111": errors.New("synthetic lookup failure")},
	}
	result, err := NewExporter(api, time.Now).Run(t.Context(), testOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Document.Participants["U11111111"].Unresolved || len(result.Warnings) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if result.Document.Messages[0].AuthorName != "U11111111 (unresolved)" {
		t.Fatalf("unresolved author name = %q", result.Document.Messages[0].AuthorName)
	}
	if !result.Manifest.Complete {
		t.Fatal("participant lookup failure must not make message export incomplete")
	}
}

func TestExporterCanExcludeThreads(t *testing.T) {
	options := testOptions(t)
	options.IncludeThreads = false
	root := slack.Message{Timestamp: "100.000001", ReplyCount: 3}
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{root}}},
		replies:      map[string][]slack.HistoryPage{},
	}
	result, err := NewExporter(api, time.Now).Run(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Manifest.Complete || result.Manifest.ThreadReplyCount != 0 {
		t.Fatalf("manifest = %#v", result.Manifest)
	}
}

func TestExporterRootRangeRetainsCompleteSelectedThreads(t *testing.T) {
	options := testOptions(t)
	options.Range = TimeRange{From: time.Unix(90, 0).UTC(), To: time.Unix(105, 0).UTC()}
	root := slack.Message{Timestamp: "100.000001", ReplyCount: 4}
	excludedRoot := slack.Message{Timestamp: "80.000001", ReplyCount: 1}
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{root, excludedRoot}}},
		replies: map[string][]slack.HistoryPage{
			root.Timestamp: {
				{
					Messages: []slack.Message{
						root,
						{Timestamp: "110.000001", Text: "after upper root boundary"},
						{Timestamp: "101.000001", Text: "inside root range"},
					},
					ResponseMetadata: slack.ResponseMetadata{NextCursor: "next"},
				},
				{
					Messages: []slack.Message{
						{Timestamp: "89.000001", Text: "before lower root boundary"},
						{Timestamp: "101.000001", Text: "inside root range"},
					},
				},
			},
		},
	}
	result, err := NewExporter(api, time.Now).Run(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.RootMessageCount != 1 || result.Manifest.ThreadReplyCount != 3 {
		t.Fatalf("manifest = %#v", result.Manifest)
	}
	got := result.Document.Messages[0].ThreadReplies
	if len(got) != 3 || got[0].Timestamp != "89.000001" || got[1].Timestamp != "101.000001" || got[2].Timestamp != "110.000001" {
		t.Fatalf("thread replies = %#v", got)
	}
	if api.replyCalls[excludedRoot.Timestamp] != 0 {
		t.Fatalf("excluded root thread was fetched %d times", api.replyCalls[excludedRoot.Timestamp])
	}
}

func TestExporterRejectsReplyFromDifferentThread(t *testing.T) {
	options := testOptions(t)
	root := slack.Message{Timestamp: "100.000001", ReplyCount: 1}
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{root}}},
		replies: map[string][]slack.HistoryPage{
			root.Timestamp: {{Messages: []slack.Message{
				root,
				{Timestamp: "101.000001", ThreadTS: "99.000001", Text: "wrong parent"},
			}}},
		},
	}
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("thread parent error = %v", err)
	}
}

func TestExporterCountsBotParticipants(t *testing.T) {
	options := testOptions(t)
	api := &fakeAPI{
		historyErrAt: -1,
		history: []slack.HistoryPage{{Messages: []slack.Message{{
			Timestamp: "100.000001",
			BotID:     "B11111111",
			Username:  "Example Bot",
		}}}},
		replies: map[string][]slack.HistoryPage{},
	}
	result, err := NewExporter(api, time.Now).Run(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.ParticipantCount != 1 || result.Document.Participants["B11111111"].DisplayName != "Example Bot" {
		t.Fatalf("participants = %#v", result.Document.Participants)
	}
}

func TestExporterRemovesFilesContainingCredentials(t *testing.T) {
	options := testOptions(t)
	secret := "generated-sensitive-" + t.Name()
	options.Secrets = []string{secret}
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{{Timestamp: "100.000001", Text: secret}}}},
		replies:      map[string][]slack.HistoryPage{},
	}

	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil || !strings.Contains(err.Error(), "credential leak") {
		t.Fatalf("error = %v", err)
	}
	for _, relative := range []string{"conversation.md", "messages_with_threads.json", "raw_history/page_0001.json"} {
		if _, err := os.Stat(filepath.Join(options.OutputDir, relative)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("unsafe file remains: %s", relative)
		}
	}
}

func TestExporterRejectsMalformedMessageTimestampAfterRawPreservation(t *testing.T) {
	options := testOptions(t)
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{{Timestamp: "invalid"}}}},
		replies:      map[string][]slack.HistoryPage{},
	}
	if _, err := NewExporter(api, time.Now).Run(t.Context(), options); err == nil || !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(options.OutputDir, "raw_history", "page_0001.json")); err != nil {
		t.Fatalf("safe raw response was not preserved: %v", err)
	}
}

func TestExporterAllowsExplicitPartialHistory(t *testing.T) {
	options := testOptions(t)
	options.AllowPartial = true
	api := &fakeAPI{
		historyErrAt: 1,
		history: []slack.HistoryPage{{
			Messages: []slack.Message{{Timestamp: "100.000001"}},
			HasMore:  true,
		}},
		replies: map[string][]slack.HistoryPage{},
	}
	result, err := NewExporter(api, time.Now).Run(t.Context(), options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.Complete || result.Manifest.RootMessageCount != 1 {
		t.Fatalf("manifest = %#v", result.Manifest)
	}
}

func TestExporterIncludesRootAndReplyReactorsAsParticipants(t *testing.T) {
	root := slack.Message{
		Timestamp:  "100.000001",
		User:       "U11111111",
		ReplyCount: 1,
		Reactions: []slack.Reaction{{
			Name:  "ok",
			Count: 2,
			Users: []string{"U22222222", "U22222222"},
		}},
	}
	reply := slack.Message{
		Timestamp: "101.000001",
		ThreadTS:  root.Timestamp,
		Reactions: []slack.Reaction{{
			Name:  "eyes",
			Count: 2,
			Users: []string{"U22222222", "U33333333"},
		}},
	}
	api := &fakeAPI{
		historyErrAt: -1,
		history:      []slack.HistoryPage{{Messages: []slack.Message{root}}},
		replies: map[string][]slack.HistoryPage{
			root.Timestamp: {{Messages: []slack.Message{root, reply}}},
		},
		users: map[string]slack.User{
			"U11111111": {ID: "U11111111", Profile: slack.UserProfile{DisplayName: "Example One"}},
			"U22222222": {ID: "U22222222", Profile: slack.UserProfile{DisplayName: "Example Two"}},
			"U33333333": {ID: "U33333333", Profile: slack.UserProfile{DisplayName: "Example Three"}},
		},
	}

	result, err := NewExporter(api, time.Now).Run(t.Context(), testOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.ParticipantCount != 3 {
		t.Fatalf("participant count = %d, want 3", result.Manifest.ParticipantCount)
	}
	if result.Manifest.SchemaVersion != 1 || result.Document.SchemaVersion != 2 {
		t.Fatalf("schema versions: manifest=%d document=%d", result.Manifest.SchemaVersion, result.Document.SchemaVersion)
	}
	for _, id := range []string{"U11111111", "U22222222", "U33333333"} {
		if _, exists := result.Document.Participants[id]; !exists {
			t.Errorf("participant %s was not resolved", id)
		}
		if api.userCalls[id] != 1 {
			t.Errorf("user lookup calls for %s = %d, want 1", id, api.userCalls[id])
		}
	}
}

func TestExporterReactionResolutionFailureWarnsWithoutIncompleteExport(t *testing.T) {
	const reactorID = "U44444444"
	api := &fakeAPI{
		historyErrAt: -1,
		history: []slack.HistoryPage{{Messages: []slack.Message{{
			Timestamp: "100.000001",
			Reactions: []slack.Reaction{{Name: "ok", Count: 1, Users: []string{reactorID}}},
		}}}},
		replies:    map[string][]slack.HistoryPage{},
		userErrors: map[string]error{reactorID: errors.New("synthetic lookup failure")},
	}

	result, err := NewExporter(api, time.Now).Run(t.Context(), testOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Document.Participants[reactorID].Unresolved {
		t.Fatalf("participant = %#v", result.Document.Participants[reactorID])
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], reactorID) {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	if !result.Manifest.Complete || result.Manifest.ParticipantCount != 1 {
		t.Fatalf("manifest = %#v", result.Manifest)
	}
}
