package app

import (
	"net/http"

	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/auth"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/googleapi"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/output"
	"github.com/Alechan/ai-resources/tools/gdrivectl/src/internal/service"
)

type Services struct {
	Doctor    *service.DoctorService
	Search    *service.SearchService
	FileMeta  *service.FileMetaService
	DocTabs   *service.DocTabsService
	DocExport *service.DocExportService
	Upload    *service.UploadService
	Output    *output.Writer
}

func NewServices(cfg Config) Services {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	tokenProvider := auth.NewGcloudTokenProvider(cfg.GcloudBin)
	driveClient := googleapi.NewDriveClient(httpClient)
	docsClient := googleapi.NewDocsClient(httpClient)

	return Services{
		Doctor:    service.NewDoctorService(cfg.GcloudBin, cfg.GcloudExists, tokenProvider, driveClient, docsClient),
		Search:    service.NewSearchService(tokenProvider, driveClient),
		FileMeta:  service.NewFileMetaService(tokenProvider, driveClient),
		DocTabs:   service.NewDocTabsService(tokenProvider, docsClient),
		DocExport: service.NewDocExportService(tokenProvider, driveClient),
		Upload:    service.NewUploadService(tokenProvider, driveClient),
		Output:    output.NewWriter(),
	}
}
