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
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/curlparse"
	exporter "github.com/Alechan/ai-resources/tools/slackctl/src/internal/export"
	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/keychain"
	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/service"
	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/slack"
)

const (
	exitOK       = 0
	exitUsage    = 2
	exitFailure  = 1
	maxCurlInput = 16 << 20
)

type slackAPI interface {
	service.DoctorClient
	exporter.API
}

type clientOptions struct {
	timeout      time.Duration
	requestDelay time.Duration
	debug        bool
	logger       func(string)
}

type application struct {
	store         auth.Store
	clientFactory func(auth.Credential, clientOptions) slackAPI
	stdin         io.Reader
	stdout        io.Writer
	stderr        io.Writer
	getenv        func(string) string
}

type commonOptions struct {
	workspace string
	timeout   time.Duration
	debug     bool
	json      bool
}

func main() {
	store := keychain.New()
	app := &application{
		store:  store,
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		getenv: os.Getenv,
		clientFactory: func(credential auth.Credential, options clientOptions) slackAPI {
			httpClient := &http.Client{Timeout: options.timeout}
			clientOptions := []slack.Option{
				slack.WithHTTPClient(httpClient),
				slack.WithRequestDelay(options.requestDelay),
			}
			if options.debug {
				clientOptions = append(clientOptions, slack.WithLogger(options.logger))
			}
			return slack.NewClient("https://"+credential.WorkspaceHost, credential, clientOptions...)
		},
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
	case "conversation":
		if len(args) < 2 || args[1] != "export" {
			err = errors.New("usage: slackctl conversation export <URL-or-ID> [options]")
		} else {
			err = a.runExport(ctx, args[2:])
		}
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
		case arg == "--debug" || arg == "--json":
			globals = append(globals, arg)
			index++
		case arg == "--workspace" || arg == "--timeout":
			if index+1 >= len(args) {
				return args
			}
			globals = append(globals, arg, args[index+1])
			index += 2
		case strings.HasPrefix(arg, "--workspace=") || strings.HasPrefix(arg, "--timeout=") ||
			strings.HasPrefix(arg, "--debug=") || strings.HasPrefix(arg, "--json="):
			globals = append(globals, arg)
			index++
		default:
			return append(append([]string(nil), args[index:]...), globals...)
		}
	}
	return args
}

func (a *application) printHelp() {
	fmt.Fprintln(a.stdout, `slackctl safely exports accessible Slack conversations.

Usage:
  slackctl init [--curl-file PATH] [--clear] [--workspace HOST]
  slackctl doctor [--workspace HOST] [--json]
  slackctl conversation export <URL-or-ID> [options]

Global options:
  --workspace HOST       Slack workspace host (or SLACKCTL_WORKSPACE)
  --timeout DURATION     request timeout (default 30s)
  --debug                redacted diagnostic output
  --json                 structured command output

Export options:
  --all | --from TIME    all history or inclusive lower bound
  --to TIME              inclusive upper bound (default now)
  --include-threads      fetch complete threads (default true)
  --format LIST          raw,json,markdown (default all)
  --output DIRECTORY     required unless --stdout
  --stdout               emit one selected normalized format
  --resolve-users        resolve participant names (default true)
  --page-size N          API page size (default 100)
  --request-delay TIME   delay between API requests (default 500ms)
  --resume               continue from raw pages
  --allow-partial        permit an incomplete export`)
}

func addCommonFlags(flags *flag.FlagSet, options *commonOptions) {
	flags.StringVar(&options.workspace, "workspace", "", "Slack workspace host")
	flags.DurationVar(&options.timeout, "timeout", 30*time.Second, "request timeout")
	flags.BoolVar(&options.debug, "debug", false, "redacted diagnostics")
	flags.BoolVar(&options.json, "json", false, "JSON output")
}

func (a *application) normalizeCommon(options *commonOptions) error {
	if options.workspace == "" {
		options.workspace = a.getenv("SLACKCTL_WORKSPACE")
	}
	if options.timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	if options.workspace != "" && !validWorkspaceHost(options.workspace) {
		return errors.New("--workspace must be a Slack workspace host")
	}
	return nil
}

func (a *application) runInit(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	curlFile := flags.String("curl-file", "", "path to copied cURL")
	clearCredential := flags.Bool("clear", false, "clear stored credentials")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := a.normalizeCommon(&common); err != nil {
		return err
	}
	if *clearCredential {
		if common.workspace == "" {
			return errors.New("--workspace is required with --clear")
		}
		if err := a.store.Clear(ctx, common.workspace); err != nil {
			return err
		}
		return a.writeCommandResult(common.json, map[string]any{"cleared": true, "workspace_host": common.workspace}, "credentials cleared: "+common.workspace)
	}
	var reader io.Reader = a.stdin
	if *curlFile != "" {
		file, err := os.Open(filepath.Clean(*curlFile))
		if err != nil {
			return errors.New("could not open cURL file")
		}
		defer file.Close()
		reader = file
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxCurlInput+1))
	if err != nil || len(data) > maxCurlInput {
		clear(data)
		return errors.New("could not read cURL input")
	}
	parsed, err := curlparse.Parse(string(data))
	clear(data)
	if err != nil {
		return err
	}
	if common.workspace != "" && common.workspace != parsed.WorkspaceHost {
		return errors.New("cURL workspace does not match --workspace")
	}
	credential := auth.Credential{WorkspaceHost: parsed.WorkspaceHost, Token: parsed.Token, Cookie: parsed.Cookie}
	client := a.newClient(credential, common, 0)
	info, err := client.AuthTest(ctx)
	if err != nil {
		return errors.New("credential validation failed")
	}
	credential.WorkspaceID = info.WorkspaceID
	if credential.WorkspaceID == "" {
		return errors.New("credential validation did not identify a workspace")
	}
	if err := a.store.Save(ctx, credential); err != nil {
		return err
	}
	return a.writeCommandResult(common.json, map[string]any{"initialized": true, "workspace_host": credential.WorkspaceHost, "workspace_id": credential.WorkspaceID}, "credentials stored in macOS Keychain: "+credential.WorkspaceHost)
}

func (a *application) runDoctor(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if err := a.normalizeCommon(&common); err != nil {
		return err
	}
	if common.workspace == "" {
		return errors.New("--workspace or SLACKCTL_WORKSPACE is required")
	}
	report, err := service.Doctor(ctx, a.store, common.workspace, func(credential auth.Credential) service.DoctorClient {
		return a.newClient(credential, common, 0)
	})
	if common.json {
		_ = writeJSON(a.stdout, report)
	} else {
		fmt.Fprintf(a.stdout, "credential store: %s\ncredentials found: %t\nworkspace reachable: %t\nauth valid: %t\nauthenticated user: %s\nconversation API: %t\n",
			report.CredentialStore, report.CredentialsFound, report.WorkspaceReachable, report.AuthValid, report.AuthenticatedUser, report.ConversationAPI)
	}
	return err
}

func (a *application) runExport(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: slackctl conversation export <URL-or-ID> [options]")
	}
	target := args[0]
	flags := flag.NewFlagSet("conversation export", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var common commonOptions
	addCommonFlags(flags, &common)
	all := flags.Bool("all", false, "all accessible history")
	from := flags.String("from", "", "inclusive start time")
	to := flags.String("to", "", "inclusive end time")
	includeThreads := flags.Bool("include-threads", true, "include thread replies")
	formatsValue := flags.String("format", "raw,json,markdown", "output formats")
	output := flags.String("output", "", "output directory")
	stdout := flags.Bool("stdout", false, "emit selected format")
	resolveUsers := flags.Bool("resolve-users", true, "resolve users")
	pageSize := flags.Int("page-size", 100, "API page size")
	requestDelay := flags.Duration("request-delay", 500*time.Millisecond, "delay between requests")
	resume := flags.Bool("resume", false, "resume export")
	allowPartial := flags.Bool("allow-partial", false, "allow incomplete export")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if err := a.normalizeCommon(&common); err != nil {
		return err
	}
	if common.workspace == "" {
		return errors.New("--workspace or SLACKCTL_WORKSPACE is required")
	}
	if *stdout && *output != "" {
		return errors.New("--stdout and --output are mutually exclusive")
	}
	if !*stdout && *output == "" {
		return errors.New("--output is required unless --stdout is used")
	}
	if *requestDelay < 0 {
		return errors.New("--request-delay must not be negative")
	}
	if *resume && *to == "" {
		data, readErr := os.ReadFile(filepath.Join(*output, "manifest.json"))
		if readErr != nil {
			return errors.New("--resume requires an existing manifest")
		}
		var previous exporter.Manifest
		if jsonErr := json.Unmarshal(data, &previous); jsonErr != nil {
			return errors.New("--resume found an invalid manifest")
		}
		*to = previous.RequestedTo
		if !*all && *from == "" {
			if previous.RequestedFrom == nil {
				*all = true
			} else {
				*from = *previous.RequestedFrom
			}
		}
	}
	timeRange, err := exporter.NormalizeRequest(exporter.RequestInput{All: *all, From: *from, To: *to}, time.Now())
	if err != nil {
		return err
	}
	formats, err := parseFormats(*formatsValue)
	if err != nil {
		return err
	}
	if *stdout && len(formats) != 1 {
		return errors.New("--stdout requires exactly one selected format")
	}
	credential, err := a.store.Load(ctx, common.workspace)
	if err != nil {
		return errors.New("credentials not found; run slackctl init")
	}
	ref, err := exporter.ParseConversation(target, credential.WorkspaceID)
	if err != nil {
		return err
	}
	client := a.newClient(credential, common, *requestDelay)
	result, err := exporter.NewExporter(client, time.Now).Run(ctx, exporter.Options{
		WorkspaceHost:  credential.WorkspaceHost,
		WorkspaceID:    credential.WorkspaceID,
		ConversationID: ref.ConversationID,
		Range:          timeRange,
		IncludeThreads: *includeThreads,
		ResolveUsers:   *resolveUsers,
		PageSize:       *pageSize,
		OutputDir:      *output,
		Formats:        formats,
		Resume:         *resume,
		AllowPartial:   *allowPartial,
		Secrets:        []string{credential.Token.Reveal(), credential.Cookie.Reveal()},
		Progress: func(message string) {
			fmt.Fprintln(a.stderr, message)
		},
		InMemory: *stdout,
	})
	if err != nil {
		return err
	}
	if result.Manifest.ConversationType != "channel" {
		fmt.Fprintln(a.stderr, "warning: export contains private conversation data; keep it local and uncommitted")
	}
	if *stdout {
		switch {
		case formats[exporter.FormatMarkdown]:
			_, err = io.WriteString(a.stdout, exporter.RenderMarkdown(result.Document))
		case formats[exporter.FormatJSON]:
			var data []byte
			data, err = exporter.MarshalDocument(result.Document)
			if err == nil {
				_, err = a.stdout.Write(data)
			}
		case formats[exporter.FormatRaw]:
			err = writeJSON(a.stdout, result.RawResponses)
		}
		return err
	}
	if common.json {
		return writeJSON(a.stdout, result.Manifest)
	}
	fmt.Fprintf(a.stdout, "export complete: %t\nroot messages: %d\nthread replies: %d\nparticipants: %d\noutput: %s\n",
		result.Manifest.Complete, result.Manifest.RootMessageCount, result.Manifest.ThreadReplyCount, result.Manifest.ParticipantCount, result.OutputDir)
	return nil
}

func (a *application) newClient(credential auth.Credential, common commonOptions, delay time.Duration) slackAPI {
	return a.clientFactory(credential, clientOptions{
		timeout:      common.timeout,
		requestDelay: delay,
		debug:        common.debug,
		logger: func(message string) {
			fmt.Fprintln(a.stderr, message)
		},
	})
}

func parseFormats(value string) (map[exporter.Format]bool, error) {
	formats := make(map[exporter.Format]bool)
	for _, item := range strings.Split(value, ",") {
		format := exporter.Format(strings.TrimSpace(item))
		switch format {
		case exporter.FormatRaw, exporter.FormatJSON, exporter.FormatMarkdown:
			formats[format] = true
		default:
			return nil, fmt.Errorf("unsupported format %q", item)
		}
	}
	if len(formats) == 0 {
		return nil, errors.New("at least one format is required")
	}
	return formats, nil
}

func validWorkspaceHost(host string) bool {
	return host != "slack.com" && strings.HasSuffix(host, ".slack.com") && !strings.ContainsAny(host, "/:@ ")
}

func containsArg(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted || strings.HasPrefix(arg, wanted+"=") {
			return true
		}
	}
	return false
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
