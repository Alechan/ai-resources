package app

import (
	"io"
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
	Monitors     *service.MonitorsService
	EventsList   *service.EventsListService
	MetricsQuery *service.MetricsQueryService
	Notebooks    *service.NotebooksService
	Dashboards   *service.DashboardsService
	Output       *output.Writer
	dd           *datadogapi.Client
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
		Monitors:     service.NewMonitorsService(ddClient, metricsSvc, cfg.Site),
		EventsList:   service.NewEventsListService(ddClient),
		MetricsQuery: metricsSvc,
		Notebooks:    service.NewNotebooksService(ddClient, metricsSvc, cfg.Site),
		Dashboards:   service.NewDashboardsService(ddClient, metricsSvc, logsSvc, cfg.Site),
		Output:       output.NewWriter(),
		dd:           ddClient,
	}
}

func (s *Services) SetDebug(w io.Writer) {
	if s.dd != nil {
		s.dd.SetDebugLogger(datadogapi.NewDebugLogger(w))
	}
}
