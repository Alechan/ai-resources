package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/fail"
)

// LogsExportInput configures a logs export.
type LogsExportInput struct {
	Query  string
	From   string
	To     string
	Limit  int
	Format string
	Fields []string
}

// WriteLogsNDJSON writes one projected event per line.
func WriteLogsNDJSON(events []LogEvent, fields []string, w io.Writer) error {
	enc := json.NewEncoder(w)
	for _, event := range events {
		if err := enc.Encode(ProjectLogEventFields(event, fields)); err != nil {
			return err
		}
	}
	return nil
}

// WriteLogsCSV writes Date,Host,Service,Content rows.
func WriteLogsCSV(events []LogEvent, fields []string, w io.Writer) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"Date", "Host", "Service", "Content"}); err != nil {
		return err
	}
	for _, event := range events {
		content, err := exportCSVContent(event, fields)
		if err != nil {
			return err
		}
		if err := writer.Write([]string{event.Timestamp, event.Host, event.Service, content}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func exportCSVContent(event LogEvent, fields []string) (string, error) {
	if len(fields) == 0 {
		if len(event.Custom) == 0 {
			return "{}", nil
		}
		b, err := json.Marshal(event.Custom)
		return string(b), err
	}
	b, err := json.Marshal(ProjectLogEventFields(event, fields))
	return string(b), err
}

// ExportLogsToFile paginates logs and writes export output.
func (s *LogsQueryService) ExportLogsToFile(ctx context.Context, input LogsExportInput, path string) (LogsQueryResult, error) {
	result, err := s.RunAll(ctx, LogsQueryInput{
		Query: input.Query,
		From:  input.From,
		To:    input.To,
		Limit: input.Limit,
	}, input.Limit)
	if err != nil {
		return LogsQueryResult{}, err
	}
	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return LogsQueryResult{}, fail.NewValidation("unable to create export directory", err.Error())
		}
		file, err := os.Create(path)
		if err != nil {
			return LogsQueryResult{}, fail.NewValidation("unable to write export output", err.Error())
		}
		defer file.Close()
		if err := writeLogsExport(result.Data, input, file); err != nil {
			return LogsQueryResult{}, err
		}
	}
	return result, nil
}

func writeLogsExport(events []LogEvent, input LogsExportInput, w io.Writer) error {
	switch input.Format {
	case "", "ndjson":
		return WriteLogsNDJSON(events, input.Fields, w)
	case "csv":
		return WriteLogsCSV(events, input.Fields, w)
	default:
		return fail.NewValidation("unsupported export format", fmt.Sprintf("use ndjson or csv, got %q", input.Format))
	}
}
