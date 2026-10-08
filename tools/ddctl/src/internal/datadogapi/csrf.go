package datadogapi

import (
	"strings"
)

func pathNeedsCSRF(path string) bool {
	if strings.Contains(path, "/api/v1/logs-analytics/list") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/dashboard") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/notebooks") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/monitor") {
		return true
	}
	return false
}
