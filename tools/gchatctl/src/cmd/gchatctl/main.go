package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/capture"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/chat"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/keychain"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/members"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/search"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/service"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/spaces"
	"github.com/Alechan/ai-resources/tools/gchatctl/src/internal/topics"
)

const (
	exitOK      = 0
	exitUsage   = 2
	exitFailure = 1
	bootstrapOp = "bootstrap"
)

var operationPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)

var errMissingCookies = errors.New(`capture has no cookies; in Chrome Network settings enable "Allow to generate HAR with sensitive data", recapture, and rerun gchatctl init`)

type application struct {
	store      auth.Store
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	getenv     func(string) string
	httpClient *http.Client
}

type commonOptions struct {
	account string
	json    bool
}

func main() {
	app := &application{
		store:  keychain.New(),
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		getenv: os.Getenv,
	}
	os.Exit(app.run(context.Background(), os.Args[1:]))
}

func (a *application) run(ctx context.Context, args []string) int {
	args = moveLeadingGlobalFlags(args)
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		a.printHelp()
		return exitOK
	}

	var err error
	jsonOutput := containsArg(args, "--json")
	switch args[0] {
	case "init":
		err = a.runInit(ctx, args[1:])
	case "doctor":
		err = a.runDoctor(ctx, args[1:])
	case "learn":
		err = a.runLearn(ctx, args[1:])
	case "fixture":
		err = a.runFixture(args[1:])
	case "topics":
		err = a.runTopics(ctx, args[1:])
	case "spaces":
		err = a.runSpaces(ctx, args[1:])
	case "members":
		err = a.runMembers(ctx, args[1:])
	case "search":
		err = a.runSearch(ctx, args[1:])
	case "conversation":
		err = a.runConversation(ctx, args[1:])
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	if err == nil {
		return exitOK
	}
	a.printError(err, jsonOutput)
	if errors.Is(err, flag.ErrHelp) || strings.HasPrefix(err.Error(), "usage:") {
		return exitUsage
	}
	return exitFailure
}

func moveLeadingGlobalFlags(args []string) []string {
	var globals []string
	index := 0
	for index < len(args) {
		arg := args[index]
		switch {
		case arg == "--json":
			globals = append(globals, arg)
			index++
		case arg == "--account":
			if index+1 >= len(args) {
				return args
			}
			globals = append(globals, arg, args[index+1])
			index += 2
		case strings.HasPrefix(arg, "--account=") || strings.HasPrefix(arg, "--json="):
			globals = append(globals, arg)
			index++
		default:
			return append(append([]string(nil), args[index:]...), globals...)
		}
	}
	return args
}

func (a *application) printHelp() {
	fmt.Fprintln(a.stdout, `gchatctl is a read-only Google Chat capture tool.

It stores a browser session from a local HAR or cURL capture and can replay
list_topics. Do not paste captures, cookies, or Authorization headers into
chat, tickets, or git.

Usage:
  gchatctl init [--har-file PATH] [--curl-file PATH] [--clear] [--account KEY]
  gchatctl doctor [--account KEY] [--live] [--json]
  gchatctl learn OPERATION [--har-file PATH] [--curl-file PATH] [--account KEY]
  gchatctl fixture [--har-file PATH] [--curl-file PATH] [--operation NAME] --out PATH
  gchatctl topics list [--space ID] [--contains TEXT] [--format LIST] [--output DIR|--stdout]
  gchatctl spaces list [--format LIST] [--output DIR|--stdout]
  gchatctl spaces get ID [--format LIST] [--output DIR|--stdout]
  gchatctl members list --space ID [--format LIST] [--output DIR|--stdout]
  gchatctl search messages --query TEXT [--format LIST] [--output DIR|--stdout]
  gchatctl conversation export <CHAT-URL> [--format LIST] [--output DIR|--stdout] [--allow-partial]

Global options:
  --account KEY   Google Chat authuser (default 0, or GCHATCTL_ACCOUNT)
  --json          structured command output

conversation export takes a chat.google.com/room permalink. The thread or
message ID is an inclusive lower bound; newer topics in that space are kept.
--space on topics list fetches that space via rewritten list_topics.
--contains filters parsed message text locally. It is not Chat search.
HAR files are streamed, so full Chrome captures larger than memory limits are accepted.`)
}

func addCommonFlags(flags *flag.FlagSet, options *commonOptions) {
	flags.StringVar(&options.account, "account", "", "Google Chat authuser key")
	flags.BoolVar(&options.json, "json", false, "JSON output")
}

func (a *application) normalizeCommon(options *commonOptions) {
	if options.account == "" {
		options.account = a.getenv("GCHATCTL_ACCOUNT")
	}
}

func (a *application) chatClient(credential auth.Credential) *chat.Client {
	return chat.NewClient(credential, a.httpClient).WithPersist(func(ctx context.Context, updated auth.Credential) error {
		return a.store.Save(ctx, updated)
	})
}

func (a *application) runInit(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	harFile := flags.String("har-file", "", "path to a saved HAR")
	curlFile := flags.String("curl-file", "", "path to copied cURL")
	clearCredential := flags.Bool("clear", false, "clear stored credentials")
	if err := flags.Parse(args); err != nil {
		return err
	}
	a.normalizeCommon(&common)
	if *clearCredential {
		if common.account == "" {
			common.account = capture.DefaultAccountKey
		}
		if err := a.store.Clear(ctx, common.account); err != nil {
			return err
		}
		return a.writeCommandResult(common.json, map[string]any{"cleared": true, "account_key": common.account}, "credentials cleared: "+common.account)
	}
	selected, _, err := a.readCapture(*harFile, *curlFile)
	if err != nil {
		return err
	}
	if selected.Cookie.Reveal() == "" {
		return errMissingCookies
	}
	if common.account != "" {
		selected.AccountKey = common.account
	}
	credential := auth.Credential{
		AccountKey:    selected.AccountKey,
		RequestHost:   selected.RequestHost,
		Method:        selected.Method,
		URLPath:       selected.URLPath,
		RawQuery:      selected.RawQuery,
		ContentType:   selected.ContentType,
		Cookie:        selected.Cookie,
		Authorization: selected.Authorization,
		ExtraHeaders:  selected.ExtraHeaders,
		Body:          selected.Body,
	}
	credential.UpsertTemplate(selected.Template(bootstrapOp))
	if err := a.store.Save(ctx, credential); err != nil {
		return err
	}
	return a.writeCommandResult(common.json, map[string]any{
		"initialized":        true,
		"account_key":        credential.AccountKey,
		"request_host":       credential.RequestHost,
		"learned_operations": credential.LearnedOperations(),
		"live_check":         "not_run",
	}, "credentials stored in macOS Keychain: "+credential.AccountKey)
}

func (a *application) runDoctor(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	live := flags.Bool("live", false, "probe Google Chat with the stored session")
	if err := flags.Parse(args); err != nil {
		return err
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	var (
		report service.DoctorReport
		err    error
	)
	if *live {
		report, err = service.DoctorLive(ctx, a.store, common.account, a.httpClient)
	} else {
		report, err = service.Doctor(ctx, a.store, common.account)
	}
	if common.json {
		_ = writeJSON(a.stdout, report)
	} else {
		fmt.Fprintf(a.stdout, "credential store: %s\ncredentials found: %t\nhost allowed: %t\ncookie present: %t\nlist_topics ready: %t\npaginated_world ready: %t\nsearch ready: %t\nlist_members ready: %t\nget_group ready: %t\nbody bytes: %d\nbody slots: %d\nbody shape: %s\nextra headers: %d\naccount: %s\nrequest host: %s\nreplay path: %s\nlearned operations: %s\nlive check: %s\n",
			report.CredentialStore, report.CredentialsFound, report.HostAllowed, report.CookiePresent, report.ListTopicsReady, report.PaginatedWorldReady, report.SearchReady, report.ListMembersReady, report.GetGroupReady, report.BodyBytes, report.BodySlots, report.BodyShape, report.ExtraHeaderCount, report.AccountKey, report.RequestHost, report.ReplayPath, strings.Join(report.LearnedOperations, ", "), report.LiveCheck)
		if *live {
			fmt.Fprintf(a.stdout, "live cookie: %t\nlive xsrf: %t\n", report.LiveCookie, report.LiveXSRF)
		}
	}
	return err
}

func (a *application) runLearn(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("learn", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	harFile := flags.String("har-file", "", "path to a saved HAR")
	curlFile := flags.String("curl-file", "", "path to copied cURL")
	operation := ""
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		operation = args[0]
		flagArgs = args[1:]
	}
	if err := flags.Parse(flagArgs); err != nil {
		return err
	}
	if operation == "" {
		if flags.NArg() != 1 {
			return errors.New("usage: gchatctl learn OPERATION [--har-file PATH] [--curl-file PATH]")
		}
		operation = flags.Arg(0)
	} else if flags.NArg() != 0 {
		return errors.New("usage: gchatctl learn OPERATION [--har-file PATH] [--curl-file PATH]")
	}
	if !operationPattern.MatchString(operation) {
		return errors.New("operation must be lowercase letters, digits, dots, underscores, or hyphens")
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	credential, err := a.store.Load(ctx, common.account)
	if err != nil {
		return errors.New("credentials not found; run gchatctl init")
	}
	selected, requests, err := a.readCapture(*harFile, *curlFile)
	if err != nil {
		return err
	}
	if matched, matchErr := capture.SelectMatching(requests, operation); matchErr == nil {
		selected = matched
	}
	template := selected.Template(operation)
	credential.UpsertTemplate(template)
	if selected.Cookie.Reveal() != "" {
		credential.Cookie = selected.Cookie
		credential.Authorization = selected.Authorization
		for name, value := range selected.ExtraHeaders {
			if credential.ExtraHeaders == nil {
				credential.ExtraHeaders = map[string]auth.Secret{}
			}
			credential.ExtraHeaders[name] = value
		}
		if strings.Contains(selected.URLPath, "list_topics") {
			credential.RequestHost = selected.RequestHost
			credential.Method = selected.Method
			credential.URLPath = selected.URLPath
			credential.RawQuery = selected.RawQuery
			credential.ContentType = selected.ContentType
			credential.Body = selected.Body
		}
	}
	if err := a.store.Save(ctx, credential); err != nil {
		return err
	}
	return a.writeCommandResult(common.json, map[string]any{
		"learned":            true,
		"account_key":        credential.AccountKey,
		"operation":          operation,
		"request_host":       selected.RequestHost,
		"url_path":           selected.URLPath,
		"learned_operations": credential.LearnedOperations(),
	}, "learned operation "+operation+" for account "+credential.AccountKey)
}

func (a *application) runFixture(args []string) error {
	flags := flag.NewFlagSet("fixture", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	harFile := flags.String("har-file", "", "path to a saved HAR")
	curlFile := flags.String("curl-file", "", "path to copied cURL")
	operation := flags.String("operation", "", "optional operation name to record")
	out := flags.String("out", "", "redacted fixture output path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("--out is required")
	}
	if *operation != "" && !operationPattern.MatchString(*operation) {
		return errors.New("operation must be lowercase letters, digits, dots, underscores, or hyphens")
	}
	selected, requests, err := a.readCapture(*harFile, *curlFile)
	if err != nil {
		return err
	}
	fixture := capture.NewFixture(requests, selected, *operation)
	if err := capture.WriteFixture(*out, fixture); err != nil {
		return errors.New("could not write fixture")
	}
	return a.writeCommandResult(common.json, map[string]any{
		"written":       true,
		"out":           *out,
		"account_key":   fixture.AccountKey,
		"request_count": len(fixture.Requests),
		"selected_host": fixture.Selected.RequestHost,
		"selected_path": fixture.Selected.URLPath,
	}, "wrote redacted fixture: "+*out)
}

func (a *application) runTopics(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprintln(a.stdout, `Usage:
  gchatctl topics list [--space ID] [--contains TEXT] [--format LIST] [--output DIR|--stdout]

--space fetches that space. Without --space, the stored list_topics body is
replayed as-is. --contains filters parsed message text on the client. It does
not call Chat search.`)
		return nil
	}
	switch args[0] {
	case "list":
		return a.runTopicsList(ctx, args[1:])
	default:
		return fmt.Errorf("unknown topics command %q", args[0])
	}
}

func (a *application) runSpaces(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprintln(a.stdout, `Usage:
  gchatctl spaces list [--format LIST] [--output DIR|--stdout]
  gchatctl spaces get ID [--format LIST] [--output DIR|--stdout]

Learn paginated_world before spaces list. Learn get_group before spaces get.`)
		return nil
	}
	switch args[0] {
	case "list":
		return a.runSpacesList(ctx, args[1:])
	case "get":
		return a.runSpacesGet(ctx, args[1:])
	default:
		return fmt.Errorf("unknown spaces command %q", args[0])
	}
}

func (a *application) runSpacesList(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("spaces list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	formatsValue := flags.String("format", "json,markdown", "json, markdown")
	output := flags.String("output", "", "output directory")
	stdout := flags.Bool("stdout", false, "emit one selected format")
	if err := flags.Parse(args); err != nil {
		return err
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	if *stdout && *output != "" {
		return errors.New("--stdout and --output are mutually exclusive")
	}
	if !*stdout && *output == "" {
		return errors.New("--output is required unless --stdout is used")
	}
	formats, err := parseFormats(*formatsValue)
	if err != nil {
		return err
	}
	if *stdout && len(formats) != 1 {
		return errors.New("--stdout requires exactly one selected format")
	}
	credential, err := a.store.Load(ctx, common.account)
	if err != nil {
		return errors.New("credentials not found; run gchatctl init")
	}
	world, err := a.chatClient(credential).PaginatedWorld(ctx)
	if err != nil {
		return chatRequestError(err)
	}
	return a.writeWorldResult(common.json, world, *stdout, *output, formats)
}

func (a *application) runSpacesGet(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("spaces get", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	spaceFlag := flags.String("space", "", "space ID")
	formatsValue := flags.String("format", "json,markdown", "json, markdown")
	output := flags.String("output", "", "output directory")
	stdout := flags.Bool("stdout", false, "emit one selected format")
	spaceID := ""
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		spaceID = strings.TrimSpace(args[0])
		flagArgs = args[1:]
	}
	if err := flags.Parse(flagArgs); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceFlag) != "" {
		spaceID = strings.TrimSpace(*spaceFlag)
	}
	if spaceID == "" || flags.NArg() != 0 {
		return errors.New("usage: gchatctl spaces get ID [--output DIR|--stdout]")
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	if *stdout && *output != "" {
		return errors.New("--stdout and --output are mutually exclusive")
	}
	if !*stdout && *output == "" {
		return errors.New("--output is required unless --stdout is used")
	}
	formats, err := parseFormats(*formatsValue)
	if err != nil {
		return err
	}
	if *stdout && len(formats) != 1 {
		return errors.New("--stdout requires exactly one selected format")
	}
	credential, err := a.store.Load(ctx, common.account)
	if err != nil {
		return errors.New("credentials not found; run gchatctl init")
	}
	group, err := a.chatClient(credential).GetGroup(ctx, spaceID)
	if err != nil {
		return chatRequestError(err)
	}
	world := spaces.World{RPC: group.RPC, Spaces: []spaces.Space{group.Space}}
	return a.writeWorldResult(common.json, world, *stdout, *output, formats)
}

func (a *application) runMembers(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprintln(a.stdout, `Usage:
  gchatctl members list --space ID [--format LIST] [--output DIR|--stdout]

Learn list_members before running this command.`)
		return nil
	}
	switch args[0] {
	case "list":
		return a.runMembersList(ctx, args[1:])
	default:
		return fmt.Errorf("unknown members command %q", args[0])
	}
}

func (a *application) runMembersList(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("members list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	spaceID := flags.String("space", "", "fetch members from this space ID")
	formatsValue := flags.String("format", "json,markdown", "json, markdown")
	output := flags.String("output", "", "output directory")
	stdout := flags.Bool("stdout", false, "emit one selected format")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*spaceID) == "" {
		return errors.New("usage: gchatctl members list --space ID [--output DIR|--stdout]")
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	if *stdout && *output != "" {
		return errors.New("--stdout and --output are mutually exclusive")
	}
	if !*stdout && *output == "" {
		return errors.New("--output is required unless --stdout is used")
	}
	formats, err := parseFormats(*formatsValue)
	if err != nil {
		return err
	}
	if *stdout && len(formats) != 1 {
		return errors.New("--stdout requires exactly one selected format")
	}
	credential, err := a.store.Load(ctx, common.account)
	if err != nil {
		return errors.New("credentials not found; run gchatctl init")
	}
	roster, err := a.chatClient(credential).ListMembers(ctx, *spaceID)
	if err != nil {
		return chatRequestError(err)
	}
	return a.writeRosterResult(common.json, roster, *spaceID, *stdout, *output, formats)
}

func (a *application) writeRosterResult(asJSON bool, roster members.Roster, spaceID string, stdout bool, output string, formats []string) error {
	ids := make([]string, 0, len(roster.Members))
	for _, member := range roster.Members {
		ids = append(ids, member.ID)
	}
	summary := map[string]any{
		"ok":           true,
		"member_count": len(roster.Members),
		"space":        spaceID,
		"member_ids":   ids,
	}
	if stdout {
		switch formats[0] {
		case "json":
			return writeJSON(a.stdout, roster)
		case "markdown":
			_, err := io.WriteString(a.stdout, members.Markdown(roster))
			return err
		default:
			return errors.New("--format must be json, markdown, or both")
		}
	}
	paths, err := members.WriteExport(output, roster, formats)
	if err != nil {
		return errors.New("could not write members export")
	}
	summary["files"] = paths
	if asJSON {
		return writeJSON(a.stdout, summary)
	}
	fmt.Fprintf(a.stdout, "listed members: %d\nspace: %s\n", len(roster.Members), spaceID)
	for _, id := range ids {
		fmt.Fprintf(a.stdout, "member: %s\n", id)
	}
	for _, path := range paths {
		fmt.Fprintln(a.stdout, "wrote:", path)
	}
	return nil
}

func (a *application) writeWorldResult(asJSON bool, world spaces.World, stdout bool, output string, formats []string) error {
	ids := make([]string, 0, len(world.Spaces))
	for _, space := range world.Spaces {
		ids = append(ids, space.ID)
	}
	summary := map[string]any{
		"ok":          true,
		"space_count": len(world.Spaces),
		"space_ids":   ids,
	}
	if stdout {
		switch formats[0] {
		case "json":
			return writeJSON(a.stdout, world)
		case "markdown":
			_, err := io.WriteString(a.stdout, spaces.Markdown(world))
			return err
		default:
			return errors.New("--format must be json, markdown, or both")
		}
	}
	paths, err := spaces.WriteExport(output, world, formats)
	if err != nil {
		return errors.New("could not write spaces export")
	}
	summary["files"] = paths
	if asJSON {
		return writeJSON(a.stdout, summary)
	}
	fmt.Fprintf(a.stdout, "listed spaces: %d\n", len(world.Spaces))
	for _, id := range ids {
		fmt.Fprintf(a.stdout, "space: %s\n", id)
	}
	for _, path := range paths {
		fmt.Fprintln(a.stdout, "wrote:", path)
	}
	return nil
}

func (a *application) runSearch(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprintln(a.stdout, `Usage:
  gchatctl search messages --query TEXT [--format LIST] [--output DIR|--stdout]

Learn search.messages before running this command. This is Chat search, not
the local --contains filter on topics list.`)
		return nil
	}
	switch args[0] {
	case "messages":
		return a.runSearchMessages(ctx, args[1:])
	default:
		return fmt.Errorf("unknown search command %q", args[0])
	}
}

func (a *application) runSearchMessages(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("search messages", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	query := flags.String("query", "", "Chat search query")
	formatsValue := flags.String("format", "json,markdown", "json, markdown")
	output := flags.String("output", "", "output directory")
	stdout := flags.Bool("stdout", false, "emit one selected format")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*query) == "" {
		return errors.New("usage: gchatctl search messages --query TEXT [--output DIR|--stdout]")
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	if *stdout && *output != "" {
		return errors.New("--stdout and --output are mutually exclusive")
	}
	if !*stdout && *output == "" {
		return errors.New("--output is required unless --stdout is used")
	}
	formats, err := parseFormats(*formatsValue)
	if err != nil {
		return err
	}
	if *stdout && len(formats) != 1 {
		return errors.New("--stdout requires exactly one selected format")
	}
	credential, err := a.store.Load(ctx, common.account)
	if err != nil {
		return errors.New("credentials not found; run gchatctl init")
	}
	result, err := a.chatClient(credential).SearchMessages(ctx, *query)
	if err != nil {
		return chatRequestError(err)
	}
	return a.writeSearchResult(common.json, result, *stdout, *output, formats)
}

func (a *application) writeSearchResult(asJSON bool, result search.Result, stdout bool, output string, formats []string) error {
	summary := map[string]any{
		"ok":        true,
		"hit_count": len(result.Hits),
		"query":     result.Query,
	}
	if stdout {
		switch formats[0] {
		case "json":
			return writeJSON(a.stdout, result)
		case "markdown":
			_, err := io.WriteString(a.stdout, search.Markdown(result))
			return err
		default:
			return errors.New("--format must be json, markdown, or both")
		}
	}
	paths, err := search.WriteExport(output, result, formats)
	if err != nil {
		return errors.New("could not write search export")
	}
	summary["files"] = paths
	if asJSON {
		return writeJSON(a.stdout, summary)
	}
	fmt.Fprintf(a.stdout, "listed hits: %d\nquery: %s\n", len(result.Hits), result.Query)
	for _, path := range paths {
		fmt.Fprintln(a.stdout, "wrote:", path)
	}
	return nil
}

func (a *application) runTopicsList(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("topics list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	spaceID := flags.String("space", "", "fetch topics from this space ID")
	contains := flags.String("contains", "", "client-side message text filter")
	formatsValue := flags.String("format", "json,markdown", "json, markdown")
	output := flags.String("output", "", "output directory")
	stdout := flags.Bool("stdout", false, "emit one selected format")
	shape := flags.Bool("shape", false, "print redacted live response structure")
	maxPages := flags.Int("max-pages", 100, "maximum list_topics pages when --space is set")
	if err := flags.Parse(args); err != nil {
		return err
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	if *shape {
		credential, err := a.store.Load(ctx, common.account)
		if err != nil {
			return errors.New("credentials not found; run gchatctl init")
		}
		_, raw, err := a.chatClient(credential).ListTopics(ctx)
		if err != nil {
			return chatRequestError(err)
		}
		_, err = fmt.Fprintln(a.stdout, topics.Shape(raw))
		return err
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	if *stdout && *output != "" {
		return errors.New("--stdout and --output are mutually exclusive")
	}
	if !*stdout && *output == "" {
		return errors.New("--output is required unless --stdout is used")
	}
	formats, err := parseFormats(*formatsValue)
	if err != nil {
		return err
	}
	if *stdout && len(formats) != 1 {
		return errors.New("--stdout requires exactly one selected format")
	}
	credential, err := a.store.Load(ctx, common.account)
	if err != nil {
		return errors.New("credentials not found; run gchatctl init")
	}
	client := a.chatClient(credential)
	var page topics.Page
	if *spaceID != "" {
		page, err = topics.ListSpace(ctx, client, topics.ListOptions{SpaceID: *spaceID, MaxPages: *maxPages})
	} else {
		page, _, err = client.ListTopics(ctx)
	}
	if err != nil {
		return chatRequestError(err)
	}
	page = topics.Filter(page, *contains)
	return a.writePageResult(common.json, page, *spaceID, *contains, *stdout, *output, formats)
}

func (a *application) runConversation(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprintln(a.stdout, `Usage:
  gchatctl conversation export <CHAT-URL> [--format LIST] [--output DIR|--stdout] [--allow-partial]

The permalink thread or message ID is an inclusive starting point. Later topics
in the same space are included. Complete selected threads are kept.`)
		return nil
	}
	switch args[0] {
	case "export":
		return a.runConversationExport(ctx, args[1:])
	default:
		return fmt.Errorf("unknown conversation command %q", args[0])
	}
}

func (a *application) runConversationExport(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gchatctl conversation export <CHAT-URL> [--output DIR|--stdout]")
	}
	target := args[0]
	flags := flag.NewFlagSet("conversation export", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	formatsValue := flags.String("format", "json,markdown", "json, markdown")
	output := flags.String("output", "", "output directory")
	stdout := flags.Bool("stdout", false, "emit one selected format")
	allowPartial := flags.Bool("allow-partial", false, "accept an incomplete export")
	maxPages := flags.Int("max-pages", 100, "maximum list_topics pages")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	a.normalizeCommon(&common)
	if common.account == "" {
		common.account = capture.DefaultAccountKey
	}
	ref, err := topics.ParsePermalink(target)
	if err != nil {
		return err
	}
	if *stdout && *output != "" {
		return errors.New("--stdout and --output are mutually exclusive")
	}
	if !*stdout && *output == "" {
		return errors.New("--output is required unless --stdout is used")
	}
	formats, err := parseFormats(*formatsValue)
	if err != nil {
		return err
	}
	if *stdout && len(formats) != 1 {
		return errors.New("--stdout requires exactly one selected format")
	}
	credential, err := a.store.Load(ctx, common.account)
	if err != nil {
		return errors.New("credentials not found; run gchatctl init")
	}
	page, err := topics.ExportFrom(ctx, a.chatClient(credential), topics.ExportOptions{
		SpaceID:      ref.SpaceID,
		StartWebID:   ref.StartWebID(),
		MaxPages:     *maxPages,
		AllowPartial: *allowPartial,
	})
	if err != nil {
		return chatRequestError(err)
	}
	return a.writePageResult(common.json, page, ref.SpaceID, "", *stdout, *output, formats)
}

func (a *application) writePageResult(asJSON bool, page topics.Page, spaceID, query string, stdout bool, output string, formats []string) error {
	topicCount, messageCount := topics.Counts(page)
	summary := map[string]any{
		"ok":            true,
		"topic_count":   topicCount,
		"message_count": messageCount,
		"space":         spaceID,
		"query":         query,
		"complete":      page.Complete,
	}
	if stdout {
		switch formats[0] {
		case "json":
			return writeJSON(a.stdout, page)
		case "markdown":
			_, err := io.WriteString(a.stdout, topics.Markdown(page, spaceID, query))
			return err
		default:
			return errors.New("--format must be json, markdown, or both")
		}
	}
	paths, err := topics.WriteExport(output, page, formats, spaceID, query)
	if err != nil {
		return errors.New("could not write topics export")
	}
	summary["files"] = paths
	if asJSON {
		return writeJSON(a.stdout, summary)
	}
	fmt.Fprintf(a.stdout, "listed topics: %d\nlisted messages: %d\ncomplete: %t\n", topicCount, messageCount, page.Complete)
	if query != "" {
		fmt.Fprintf(a.stdout, "query: %s\n", query)
	}
	if spaceID != "" {
		fmt.Fprintf(a.stdout, "space: %s\n", spaceID)
	}
	for _, path := range paths {
		fmt.Fprintln(a.stdout, "wrote:", path)
	}
	return nil
}

func parseFormats(value string) ([]string, error) {
	seen := map[string]bool{}
	var formats []string
	for _, part := range strings.Split(value, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		if name == "" {
			continue
		}
		if name != "json" && name != "markdown" {
			return nil, errors.New("--format must be json, markdown, or both")
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		formats = append(formats, name)
	}
	if len(formats) == 0 {
		return nil, errors.New("--format must be json, markdown, or both")
	}
	return formats, nil
}

func (a *application) readCapture(harFile, curlFile string) (capture.Request, []capture.Request, error) {
	if harFile != "" && curlFile != "" {
		return capture.Request{}, nil, errors.New("--har-file and --curl-file are mutually exclusive")
	}
	var reader io.Reader = a.stdin
	switch {
	case harFile != "":
		file, err := os.Open(filepath.Clean(harFile))
		if err != nil {
			return capture.Request{}, nil, errors.New("could not open HAR file")
		}
		defer file.Close()
		reader = file
	case curlFile != "":
		file, err := os.Open(filepath.Clean(curlFile))
		if err != nil {
			return capture.Request{}, nil, errors.New("could not open cURL file")
		}
		defer file.Close()
		reader = file
	}
	requests, err := capture.ParseAllReader(reader)
	if err != nil {
		return capture.Request{}, nil, err
	}
	selected, err := capture.Select(requests)
	if err != nil {
		return capture.Request{}, nil, err
	}
	return selected, requests, nil
}

func containsArg(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted || strings.HasPrefix(arg, wanted+"=") {
			return true
		}
	}
	return false
}

func chatRequestError(err error) error {
	var api *chat.APIError
	if !errors.As(err, &api) {
		return err
	}
	if api.Kind == chat.ErrorAuth {
		return fmt.Errorf("Google Chat rejected the stored session (HTTP %d); copy a fresh list_topics cURL and rerun gchatctl init", api.StatusCode)
	}
	return fmt.Errorf("Google Chat API error (HTTP %d)", api.StatusCode)
}

func (a *application) writeCommandResult(asJSON bool, value any, text string) error {
	if asJSON {
		return writeJSON(a.stdout, value)
	}
	_, err := fmt.Fprintln(a.stdout, text)
	return err
}

func (a *application) printError(err error, asJSON bool) {
	if asJSON {
		_ = writeJSON(a.stderr, map[string]any{
			"ok": false,
			"error": map[string]string{
				"kind":    "command_failed",
				"message": err.Error(),
			},
		})
		return
	}
	fmt.Fprintln(a.stderr, "error:", err)
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
