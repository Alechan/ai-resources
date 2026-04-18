package app

import (
	"net/http"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/output"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

type Services struct {
	Auth          *auth.KeychainProvider
	Doctor        *service.DoctorService
	LogsQuery     *service.LogsQueryService
	MonitorsList  *service.MonitorsListService
	MonitorsGet   *service.MonitorsGetService
	EventsList    *service.EventsListService
	MetricsQuery  *service.MetricsQueryService
	Output        *output.Writer
}

func NewServices(cfg Config) Services {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	authProvider := auth.NewKeychainProvider(cfg.Site)
	ddClient := datadogapi.NewClient(httpClient, cfg.Site, authProvider)

	return Services{
		Auth:         authProvider,
		Doctor:       service.NewDoctorService(authProvider, ddClient),
		LogsQuery:    service.NewLogsQueryService(ddClient),
		MonitorsList: service.NewMonitorsListService(ddClient),
		MonitorsGet:  service.NewMonitorsGetService(ddClient),
		EventsList:   service.NewEventsListService(ddClient),
		MetricsQuery: service.NewMetricsQueryService(ddClient),
		Output:       output.NewWriter(),
	}
}
