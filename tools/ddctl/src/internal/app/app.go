package app

import (
	"net/http"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/output"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

type Services struct {
	Doctor    *service.DoctorService
	LogsQuery *service.LogsQueryService
	Output    *output.Writer
}

func NewServices(cfg Config) Services {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	cookieProvider := auth.NewChromeCookieProvider(cfg.CookiesPath)
	ddClient := datadogapi.NewClient(httpClient, cfg.Site, cookieProvider)

	return Services{
		Doctor:    service.NewDoctorService(cookieProvider, ddClient),
		LogsQuery: service.NewLogsQueryService(ddClient),
		Output:    output.NewWriter(),
	}
}
