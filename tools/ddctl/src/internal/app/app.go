package app

import (
	"net/http"

	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/datadogapi"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/output"
	"github.com/Alechan/ai-resources/tools/ddctl/src/internal/service"
)

type Services struct {
	Auth         *auth.KeychainProvider
	Doctor       *service.DoctorService
	LogsQuery    *service.LogsQueryService
	MonitorsList *service.MonitorsListService
	MonitorsGet  *service.MonitorsGetService
	EventsList   *service.EventsListService
	MetricsQuery *service.MetricsQueryService
	Notebooks    *service.NotebooksService
	Dashboards   *service.DashboardsService
	Output       *output.Writer
}

func NewServices(cfg Config) Services {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	authProvider := auth.NewKeychainProvider(cfg.Site)
	ddClient := datadogapi.NewClient(httpClient, cfg.Site, authProvider)
	metricsSvc := service.NewMetricsQueryService(ddClient)
	logsSvc := service.NewLogsQueryService(ddClient)

	return Services{
		Auth:         authProvider,
		Doctor:       service.NewDoctorService(authProvider, ddClient),
		LogsQuery:    logsSvc,
		MonitorsList: service.NewMonitorsListService(ddClient, cfg.Site),
		MonitorsGet:  service.NewMonitorsGetService(ddClient, cfg.Site),
		EventsList:   service.NewEventsListService(ddClient),
		MetricsQuery: metricsSvc,
		Notebooks:    service.NewNotebooksService(ddClient, metricsSvc),
		Dashboards:   service.NewDashboardsService(ddClient, metricsSvc, logsSvc, cfg.Site),
		Output:       output.NewWriter(),
	}
}
