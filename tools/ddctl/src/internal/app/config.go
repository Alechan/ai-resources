package app

import "time"

type Config struct {
	Site    string
	Timeout time.Duration
	JSON    bool
	Debug   bool
}

func NewConfig(site string, timeout time.Duration, json, debug bool) Config {
	return Config{
		Site:    site,
		Timeout: timeout,
		JSON:    json,
		Debug:   debug,
	}
}
