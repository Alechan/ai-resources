package app

import "time"

type Config struct {
	GcloudBin    string
	GcloudExists bool
	Timeout      time.Duration
	JSON         bool
	Debug        bool
}

func NewConfig(gcloudBin string, gcloudExists bool, timeout time.Duration, json, debug bool) Config {
	return Config{
		GcloudBin:    gcloudBin,
		GcloudExists: gcloudExists,
		Timeout:      timeout,
		JSON:         json,
		Debug:        debug,
	}
}
