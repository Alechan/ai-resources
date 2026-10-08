package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/capture"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/chat"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

type DoctorReport struct {
	CredentialStore     string   `json:"credential_store"`
	CredentialsFound    bool     `json:"credentials_found"`
	HostAllowed         bool     `json:"host_allowed"`
	CookiePresent       bool     `json:"cookie_present"`
	ListTopicsReady     bool     `json:"list_topics_ready"`
	PaginatedWorldReady bool     `json:"paginated_world_ready"`
	SearchReady         bool     `json:"search_ready"`
	ListMembersReady    bool     `json:"list_members_ready"`
	GetGroupReady       bool     `json:"get_group_ready"`
	BodyBytes           int      `json:"body_bytes,omitempty"`
	BodySlots           int      `json:"body_slots,omitempty"`
	BodyShape           string   `json:"body_shape,omitempty"`
	ExtraHeaderCount    int      `json:"extra_header_count,omitempty"`
	AccountKey          string   `json:"account_key,omitempty"`
	RequestHost         string   `json:"request_host,omitempty"`
	ReplayPath          string   `json:"replay_path,omitempty"`
	LearnedOperations   []string `json:"learned_operations"`
	LiveCheck           string   `json:"live_check"`
	LiveCookie          bool     `json:"live_cookie,omitempty"`
	LiveXSRF            bool     `json:"live_xsrf,omitempty"`
}

func Doctor(ctx context.Context, store auth.Store, accountKey string) (DoctorReport, error) {
	report := DoctorReport{
		CredentialStore:   "macOS Keychain (gchatctl / " + accountKey + ")",
		LearnedOperations: []string{},
		LiveCheck:         "not_run",
	}
	credential, err := store.Load(ctx, accountKey)
	if err != nil {
		return report, errors.New("credentials not found; run gchatctl init")
	}
	report.CredentialsFound = true
	report.AccountKey = credential.AccountKey
	report.RequestHost = credential.RequestHost
	report.HostAllowed = capture.AllowedHost(credential.RequestHost)
	report.CookiePresent = credential.Cookie.Reveal() != ""
	report.ReplayPath = credential.URLPath
	report.ListTopicsReady = strings.Contains(credential.URLPath, "list_topics") && credential.Body.Reveal() != ""
	if template, ok := credential.OperationTemplate("paginated_world"); ok {
		report.PaginatedWorldReady = strings.Contains(template.URLPath, "paginated_world") && template.Body != ""
	}
	if template, ok := credential.OperationTemplate("search.messages"); ok {
		report.SearchReady = strings.Contains(strings.ToLower(template.URLPath+template.RawQuery), "batchexecute") && strings.Contains(template.Body, "f.req")
	}
	if template, ok := credential.OperationTemplate("list_members"); ok {
		report.ListMembersReady = strings.Contains(template.URLPath, "list_members") && template.Body != ""
	}
	if template, ok := credential.OperationTemplate("get_group"); ok {
		report.GetGroupReady = strings.Contains(template.URLPath, "get_group") && template.Body != ""
	}
	report.BodyBytes = len(credential.Body.Reveal())
	report.BodySlots = jsonArrayLen(credential.Body.Reveal())
	if report.BodySlots > 0 {
		report.BodyShape = topics.SlotShape([]byte(credential.Body.Reveal()))
	}
	report.ExtraHeaderCount = len(credential.ExtraHeaders)
	report.LearnedOperations = credential.LearnedOperations()
	if report.LearnedOperations == nil {
		report.LearnedOperations = []string{}
	}
	if !report.HostAllowed {
		return report, errors.New("stored request host is not a Google Chat host")
	}
	if !report.CookiePresent {
		return report, errors.New("stored credentials do not include a cookie")
	}
	return report, nil
}

func DoctorLive(ctx context.Context, store auth.Store, accountKey string, client *http.Client) (DoctorReport, error) {
	report, err := Doctor(ctx, store, accountKey)
	credential, loadErr := store.Load(ctx, accountKey)
	if loadErr != nil {
		return report, errors.New("credentials not found; run gchatctl init")
	}
	if err != nil && !(isLoopbackHost(credential.RequestHost) && !report.HostAllowed && report.CookiePresent) {
		return report, err
	}
	report.LiveCookie = credential.Cookie.Reveal() != ""
	report.LiveXSRF = headerPresent(credential, "x-framework-xsrf-token")
	if !report.ListTopicsReady {
		report.LiveCheck = "skipped"
		return report, nil
	}
	probe := chat.NewClient(credential, client).WithPersist(func(ctx context.Context, updated auth.Credential) error {
		return store.Save(ctx, updated)
	})
	_, _, probeErr := probe.ListTopics(ctx)
	report.LiveCheck = chat.ProbeStatus(probeErr)
	return report, nil
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(host)
	if h, _, ok := strings.Cut(host, ":"); ok {
		host = h
	}
	return host == "127.0.0.1" || host == "localhost"
}

func headerPresent(credential auth.Credential, name string) bool {
	for key, value := range credential.ExtraHeaders {
		if strings.EqualFold(key, name) && value.Reveal() != "" {
			return true
		}
	}
	return false
}

func jsonArrayLen(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '[' {
		return 0
	}
	var body []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return 0
	}
	return len(body)
}
