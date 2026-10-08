package main

import (
	"flag"
	"io"
)

type logsQueryFlags struct {
	query       string
	from        string
	to          string
	limit       int
	all         bool
	countOnly   bool
	cursor      string
	fields      string
	verbose     bool
	verboseKeys string
}

type logsExportFlags struct {
	logsQueryFlags
	format string
	output string
}

func registerLogsQueryFlags(fs *flag.FlagSet, flags *logsQueryFlags) {
	fs.StringVar(&flags.query, "query", "*", "search query string")
	fs.String("q", "*", "search query string (shorthand)")
	fs.StringVar(&flags.from, "from", "now-1h", "start time (relative or ISO-8601)")
	fs.StringVar(&flags.to, "to", "now", "end time (relative or ISO-8601)")
	fs.IntVar(&flags.limit, "limit", 50, "max results per page (1-1000); when --all is set, max total results")
	fs.BoolVar(&flags.all, "all", false, "auto-paginate until no more results or --limit is reached")
	fs.BoolVar(&flags.countOnly, "count-only", false, "return only metadata and total hit count")
	fs.StringVar(&flags.cursor, "cursor", "", "pagination cursor from a previous result's next_cursor field")
	fs.StringVar(&flags.fields, "fields", "", "comma-separated JSON field projection (JSON mode only)")
	fs.BoolVar(&flags.verbose, "verbose", false, "include selected custom fields in text output")
	fs.Bool("v", false, "alias for --verbose")
	fs.StringVar(&flags.verboseKeys, "verbose-keys", "", "comma-separated custom keys for --verbose text output")
}

func parseLogsQueryFlags(fs *flag.FlagSet, args []string) (logsQueryFlags, error) {
	fs.SetOutput(io.Discard)
	var flags logsQueryFlags
	registerLogsQueryFlags(fs, &flags)
	if err := fs.Parse(args); err != nil {
		return logsQueryFlags{}, err
	}
	if q := fs.Lookup("q"); q != nil && q.Value.String() != "*" {
		flags.query = q.Value.String()
	}
	if v := fs.Lookup("v"); v != nil && v.Value.String() == "true" {
		flags.verbose = true
	}
	return flags, nil
}

func parseLogsExportFlags(args []string) (logsExportFlags, error) {
	fs := flag.NewFlagSet("logs export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var flags logsExportFlags
	registerLogsQueryFlags(fs, &flags.logsQueryFlags)
	fs.StringVar(&flags.format, "format", "ndjson", "export format: ndjson or csv")
	fs.StringVar(&flags.output, "output", "", "output file path")
	fs.String("o", "", "output file path")
	if err := fs.Parse(args); err != nil {
		return logsExportFlags{}, err
	}
	if q := fs.Lookup("q"); q != nil && q.Value.String() != "*" {
		flags.query = q.Value.String()
	}
	if o := fs.Lookup("o"); o != nil && o.Value.String() != "" {
		flags.output = o.Value.String()
	}
	if flags.output == "" {
		flags.output = "-"
	}
	if flags.limit == 50 && !flagProvided(fs, "limit") {
		flags.limit = 1000
	}
	return flags, nil
}

func flagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
