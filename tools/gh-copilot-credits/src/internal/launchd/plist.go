package launchd

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"time"
)

const DefaultLabel = "com.alejandro.gh-copilot-credits"

type Options struct {
	Label           string
	BinaryPath      string
	CSVPath         string
	Interval        time.Duration
	StdoutPath      string
	StderrPath      string
	EnvironmentPath string
}

func ParseInterval(value string) (time.Duration, error) {
	interval, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid interval %q: %w", value, err)
	}
	if interval <= 0 || interval > 24*time.Hour || interval%time.Hour != 0 {
		return 0, errors.New("interval must be a positive whole number of hours no greater than 24h")
	}
	hours := int(interval / time.Hour)
	if 24%hours != 0 {
		return 0, errors.New("interval must divide evenly into 24 hours")
	}
	return interval, nil
}

func GeneratePlist(options Options) ([]byte, error) {
	if options.Label == "" || options.BinaryPath == "" || options.CSVPath == "" {
		return nil, errors.New("label, binary path, and CSV path are required")
	}
	if options.Interval <= 0 {
		return nil, errors.New("interval must be positive")
	}
	hours := int(options.Interval / time.Hour)
	if options.Interval%time.Hour != 0 || hours <= 0 || hours > 24 || 24%hours != 0 {
		return nil, errors.New("interval must be a positive whole number of hours dividing 24 hours")
	}

	var builder bytes.Buffer
	builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	builder.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	builder.WriteString("<plist version=\"1.0\"><dict>\n")
	writeString(&builder, "Label", options.Label)
	builder.WriteString("<key>ProgramArguments</key><array>\n")
	for _, arg := range []string{options.BinaryPath, "current", "append", "--csv", options.CSVPath} {
		builder.WriteString("<string>")
		xml.EscapeText(&builder, []byte(arg))
		builder.WriteString("</string>\n")
	}
	builder.WriteString("</array>\n")
	builder.WriteString("<key>StartCalendarInterval</key><array>\n")
	for hour := 0; hour < 24; hour += hours {
		fmt.Fprintf(&builder, "<dict><key>Hour</key><integer>%d</integer><key>Minute</key><integer>0</integer></dict>\n", hour)
	}
	builder.WriteString("</array>\n")
	if options.StdoutPath != "" {
		writeString(&builder, "StandardOutPath", options.StdoutPath)
	}
	if options.StderrPath != "" {
		writeString(&builder, "StandardErrorPath", options.StderrPath)
	}
	if options.EnvironmentPath != "" {
		builder.WriteString("<key>EnvironmentVariables</key><dict>\n")
		writeString(&builder, "PATH", options.EnvironmentPath)
		builder.WriteString("</dict>\n")
	}
	builder.WriteString("<key>RunAtLoad</key><false/>\n")
	builder.WriteString("</dict></plist>\n")
	return builder.Bytes(), nil
}

func writeString(builder *bytes.Buffer, key, value string) {
	builder.WriteString("<key>")
	xml.EscapeText(builder, []byte(key))
	builder.WriteString("</key><string>")
	xml.EscapeText(builder, []byte(value))
	builder.WriteString("</string>\n")
}
