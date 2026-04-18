package app

import "time"

type Config struct {
	CookiesPath string
	Site        string
	Timeout     time.Duration
	JSON        bool
	Debug       bool
}

func NewConfig(cookiesPath, site string, timeout time.Duration, json, debug bool) Config {
	return Config{
		CookiesPath: cookiesPath,
		Site:        site,
		Timeout:     timeout,
		JSON:        json,
		Debug:       debug,
	}
}
