package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/copilot"
	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/csvlog"
	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/forecast"
	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/launchd"
	"github.com/Alechan/ai-resources/tools/gh-copilot-credits/src/internal/output"
)

const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

func main() {
	client := copilot.NewGHClientWithCommand(copilot.ExecCommandRunner{}, 15*time.Second, copilot.ResolveGHCommand())
	os.Exit(run(os.Args[1:], client, os.Stdout, os.Stderr, time.Now))
}

func run(args []string, client copilot.Client, stdout, stderr io.Writer, now func() time.Time) int {
	return runWithLookPath(args, client, stdout, stderr, now, exec.LookPath)
}

func runWithLookPath(args []string, client copilot.Client, stdout, stderr io.Writer, now func() time.Time, lookPath func(string) (string, error)) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printHelp(stdout)
		return exitOK
	}
	if args[0] == "doctor" {
		return runDoctor(client, stdout, stderr, lookPath)
	}
	if args[0] == "schedule" {
		return runSchedule(args[1:], stdout, stderr, lookPath)
	}
	if args[0] != "current" {
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return exitUsage
	}
	if len(args) > 1 && args[1] == "append" {
		return runCurrentAppend(args[2:], client, stdout, stderr, now)
	}
	return runCurrent(args[1:], client, stdout, stderr, now)
}

func runSchedule(args []string, stdout, stderr io.Writer, lookPath func(string) (string, error)) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "schedule: subcommand is required (print, install, status, uninstall)")
		return exitUsage
	}
	command := args[0]
	flags := flag.NewFlagSet("schedule "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	csvPath := flags.String("csv", defaultCSVPath(), "CSV snapshot path")
	intervalText := flags.String("interval", "6h", "polling interval")
	force := flags.Bool("force", false, "replace an existing LaunchAgent")
	yes := flags.Bool("yes", false, "confirm uninstall")
	if err := flags.Parse(args[1:]); err != nil {
		fmt.Fprintf(stderr, "schedule %s: %v\n", command, err)
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "schedule %s: unexpected argument %q\n", command, flags.Arg(0))
		return exitUsage
	}

	interval, err := launchd.ParseInterval(*intervalText)
	if command == "status" || command == "uninstall" {
		interval = 6 * time.Hour
		err = nil
	}
	if err != nil {
		fmt.Fprintf(stderr, "schedule %s: %v\n", command, err)
		return exitUsage
	}
	plistPath, logOut, logErr, err := schedulePaths()
	if err != nil {
		fmt.Fprintf(stderr, "schedule %s: %v\n", command, err)
		return exitFailure
	}
	label := launchd.DefaultLabel
	domain := fmt.Sprintf("gui/%d", os.Getuid())

	switch command {
	case "print":
		ghPath, err := lookPath("gh")
		if err != nil {
			fmt.Fprintf(stderr, "schedule print: gh not found\n")
			return exitFailure
		}
		return schedulePrint(stdout, stderr, label, *csvPath, interval, logOut, logErr, ghEnvironmentPath(ghPath))
	case "install":
		ghPath, err := lookPath("gh")
		if err != nil {
			fmt.Fprintf(stderr, "schedule install: gh not found\n")
			return exitFailure
		}
		return scheduleInstall(stdout, stderr, label, domain, plistPath, *csvPath, interval, logOut, logErr, ghEnvironmentPath(ghPath), *force)
	case "status":
		return scheduleStatus(stdout, label, domain, plistPath, *csvPath, *intervalText)
	case "uninstall":
		return scheduleUninstall(stdout, stderr, label, domain, plistPath, *yes)
	default:
		fmt.Fprintf(stderr, "unknown schedule subcommand %q\n", command)
		return exitUsage
	}
}

func schedulePrint(stdout, stderr io.Writer, label, csvPath string, interval time.Duration, logOut, logErr, environmentPath string) int {
	binary, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "schedule print: resolve executable: %v\n", err)
		return exitFailure
	}
	plist, err := launchd.GeneratePlist(launchd.Options{
		Label: label, BinaryPath: binary, CSVPath: csvPath, Interval: interval,
		StdoutPath: logOut, StderrPath: logErr,
		EnvironmentPath: environmentPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "schedule print: %v\n", err)
		return exitFailure
	}
	_, _ = stdout.Write(plist)
	return exitOK
}

func scheduleInstall(stdout, stderr io.Writer, label, domain, plistPath, csvPath string, interval time.Duration, logOut, logErr, environmentPath string, force bool) int {
	if _, err := os.Stat(plistPath); err == nil && !force {
		fmt.Fprintf(stderr, "schedule install: %s already exists; use --force to replace it\n", plistPath)
		return exitFailure
	}
	if force {
		_ = launchd.Bootout(context.Background(), launchd.ExecCommandRunner{}, domain+"/"+label)
	}
	binary, err := os.Executable()
	if err != nil {
		fmt.Fprintf(stderr, "schedule install: resolve executable: %v\n", err)
		return exitFailure
	}
	plist, err := launchd.GeneratePlist(launchd.Options{
		Label: label, BinaryPath: binary, CSVPath: csvPath, Interval: interval,
		StdoutPath: logOut, StderrPath: logErr,
		EnvironmentPath: environmentPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "schedule install: %v\n", err)
		return exitFailure
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o700); err != nil {
		fmt.Fprintf(stderr, "schedule install: create LaunchAgents directory: %v\n", err)
		return exitFailure
	}
	if err := os.MkdirAll(filepath.Dir(logOut), 0o700); err != nil {
		fmt.Fprintf(stderr, "schedule install: create log directory: %v\n", err)
		return exitFailure
	}
	if err := os.WriteFile(plistPath, plist, 0o600); err != nil {
		fmt.Fprintf(stderr, "schedule install: write plist: %v\n", err)
		return exitFailure
	}
	if err := launchd.Bootstrap(context.Background(), launchd.ExecCommandRunner{}, domain, plistPath); err != nil {
		fmt.Fprintf(stderr, "schedule install: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "installed %s\n", plistPath)
	return exitOK
}

func scheduleStatus(stdout io.Writer, label, domain, plistPath, csvPath, interval string) int {
	_, fileErr := os.Stat(plistPath)
	loadedErr := launchd.Print(context.Background(), launchd.ExecCommandRunner{}, domain+"/"+label)
	installed := fileErr == nil
	loaded := loadedErr == nil
	fmt.Fprintf(stdout, "LaunchAgent: %s\n", map[bool]string{true: "installed", false: "not installed"}[installed])
	fmt.Fprintf(stdout, "Loaded:      %s\n", map[bool]string{true: "yes", false: "no"}[loaded])
	fmt.Fprintf(stdout, "Label:       %s\n", label)
	fmt.Fprintf(stdout, "Plist:       %s\n", plistPath)
	fmt.Fprintf(stdout, "CSV:         %s\n", csvPath)
	fmt.Fprintf(stdout, "Interval:    %s\n", interval)
	if !installed && !loaded {
		return exitFailure
	}
	return exitOK
}

func scheduleUninstall(stdout, stderr io.Writer, label, domain, plistPath string, confirmed bool) int {
	if !confirmed {
		fmt.Fprintln(stderr, "schedule uninstall: --yes is required")
		return exitUsage
	}
	if err := launchd.Bootout(context.Background(), launchd.ExecCommandRunner{}, domain+"/"+label); err != nil {
		if _, statErr := os.Stat(plistPath); statErr != nil {
			fmt.Fprintf(stderr, "schedule uninstall: %v\n", err)
			return exitFailure
		}
	}
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "schedule uninstall: remove plist: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "uninstalled %s\n", label)
	return exitOK
}

func schedulePaths() (plistPath, logOut, logErr string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", err
	}
	logs := filepath.Join(home, "Library", "Logs", "gh-copilot-credits")
	return filepath.Join(home, "Library", "LaunchAgents", launchd.DefaultLabel+".plist"), filepath.Join(logs, "stdout.log"), filepath.Join(logs, "stderr.log"), nil
}

func defaultCSVPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "usage.csv"
	}
	return filepath.Join(home, "Library", "Application Support", "gh-copilot-credits", "usage.csv")
}

func ghEnvironmentPath(ghPath string) string {
	directory := filepath.Dir(ghPath)
	return directory + ":/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
}

func runDoctor(client copilot.Client, stdout, stderr io.Writer, lookPath func(string) (string, error)) int {
	path, err := lookPath("gh")
	if err != nil {
		fmt.Fprintf(stderr, "gh: not found\n")
		return exitFailure
	}
	fmt.Fprintf(stdout, "gh: found (%s)\n", path)
	data, err := client.Current(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "endpoint: %v\n", err)
		return exitFailure
	}
	quota, err := data.PremiumQuota()
	if err != nil {
		fmt.Fprintf(stderr, "quota: %v\n", err)
		return exitFailure
	}
	fmt.Fprintln(stdout, "endpoint: ok")
	fmt.Fprintf(stdout, "quota: %s\n", quota.QuotaID)
	fmt.Fprintf(stdout, "reset: %s\n", data.QuotaResetDateUTC)
	return exitOK
}

func runCurrent(args []string, client copilot.Client, stdout, stderr io.Writer, now func() time.Time) int {
	flags := flag.NewFlagSet("current", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	jsonOutput := flags.Bool("json", false, "emit filtered JSON")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(stderr, "current: %v\n", err)
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "current: unexpected argument %q\n", flags.Arg(0))
		return exitUsage
	}

	data, err := client.Current(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return exitFailure
	}
	quota, err := data.PremiumQuota()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return exitFailure
	}
	observedAt := now()
	resetDate, err := time.Parse(time.RFC3339, data.QuotaResetDateUTC)
	if err != nil {
		fmt.Fprintf(stderr, "invalid quota reset date: %v\n", err)
		return exitFailure
	}
	used := quota.DerivedUsedCredits()
	if quota.CreditsUsed != nil {
		used = *quota.CreditsUsed
	}
	forecastReport, err := forecast.Calculate(used, quota.Entitlement, resetDate, observedAt)
	if err != nil {
		fmt.Fprintf(stderr, "calculate forecast: %v\n", err)
		return exitFailure
	}
	report := output.CurrentReport{
		ObservedAtUTC:     observedAt.UTC().Format(time.RFC3339),
		APITimestampUTC:   quota.TimestampUTC,
		CopilotPlan:       data.CopilotPlan,
		QuotaResetDateUTC: data.QuotaResetDateUTC,
		Quota:             quota,
		Forecast:          &forecastReport,
	}

	if *jsonOutput {
		encoded, err := output.RenderCurrentJSON(report)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return exitFailure
		}
		_, _ = io.WriteString(stdout, encoded)
		return exitOK
	}
	_, _ = io.WriteString(stdout, output.RenderCurrentTable(report))
	return exitOK
}

func runCurrentAppend(args []string, client copilot.Client, stdout, stderr io.Writer, now func() time.Time) int {
	flags := flag.NewFlagSet("current append", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	csvPath := flags.String("csv", "", "CSV snapshot path")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(stderr, "current append: %v\n", err)
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "current append: unexpected argument %q\n", flags.Arg(0))
		return exitUsage
	}
	if *csvPath == "" {
		fmt.Fprintln(stderr, "current append: --csv is required")
		return exitUsage
	}

	data, err := client.Current(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return exitFailure
	}
	quota, err := data.PremiumQuota()
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return exitFailure
	}
	if err := csvlog.Append(*csvPath, csvlog.SnapshotFrom(data, quota, now())); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "appended snapshot to %s\n", *csvPath)
	return exitOK
}

func printHelp(writer io.Writer) {
	fmt.Fprintln(writer, `gh-copilot-credits reads GitHub Copilot AI-credit quota snapshots.

Usage:
  gh-copilot-credits current [--json]
  gh-copilot-credits current append --csv PATH
  gh-copilot-credits doctor
  gh-copilot-credits schedule print|install|status|uninstall

Authentication is delegated to the local gh CLI. Raw API responses and
credentials are never written by this tool.`)
}

func printUsage(writer io.Writer) {
	text := "usage: gh-copilot-credits current [--json]"
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	_, _ = io.WriteString(writer, text)
}
