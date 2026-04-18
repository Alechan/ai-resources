package app

import (
	"net/http"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/output"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

type Services struct {
	Auth      *auth.KeychainProvider
	Doctor    *service.DoctorService
	LogsQuery *service.LogsQueryService
	Output    *output.Writer
}

func NewServices(cfg Config) Services {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	authProvider := auth.NewKeychainProvider(cfg.Site)
	ddClient := datadogapi.NewClient(httpClient, cfg.Site, authProvider)

	return Services{
		Auth:      authProvider,
		Doctor:    service.NewDoctorService(authProvider, ddClient),
		LogsQuery: service.NewLogsQueryService(ddClient),
		Output:    output.NewWriter(),
	}
}
