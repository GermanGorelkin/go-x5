package logistics

import (
	"fmt"
	"io"

	"go.uber.org/zap"
)

type SalesChannel string
type TypeReport string
type ReportStatus string

const (
	TS5   SalesChannel = "TS5" // Pyaterochka
	TSX   SalesChannel = "TSX" // Perekrestok
	TSK   SalesChannel = "TSK" // Karusel
	TSALL SalesChannel = "ALL" // All channels

	SALES             TypeReport = "SALES"             // sales report
	INVENTORY         TypeReport = "INVENTORY"         // inventory report
	MOVEMENT          TypeReport = "MOVEMENT"          // write-off report
	CHECK             TypeReport = "CHECK"             // all channels
	PRODUCT_DIRECTORY TypeReport = "PRODUCT_DIRECTORY" // all channels
	SHOP_DIRECTORY    TypeReport = "SHOP_DIRECTORY"    // all channels

	CREATED              ReportStatus = "CREATED"              // request created
	BUILD                ReportStatus = "BUILD"                // report is being generated
	DONE                 ReportStatus = "DONE"                 // report ready
	ERROR                ReportStatus = "ERROR"                // report generation error
	DOWNLOADED           ReportStatus = "DOWNLOADED"           // report downloaded
	REMOVAL_EXPIRED_TIME ReportStatus = "REMOVAL_EXPIRED_TIME" // removed due to expiration
	REMOVAL_MANUAL       ReportStatus = "REMOVAL_MANUAL"       // removed manually by admin
)

// ReportService handles report creation, status polling, and downloading.
type ReportService service

// RequestCreateReport holds the parameters for creating a new report request.
type RequestCreateReport struct {
	FinishDate   string       `json:"finishDate"`
	StartDate    string       `json:"startDate"`
	SalesChannel SalesChannel `json:"salesChannel"`
	TypeReport   TypeReport   `json:"typeReport"`
	IsArchive    bool         `json:"isArchive"`
}

// ResponseCreateReport is the API response returned after a report creation request.
type ResponseCreateReport struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Result      struct {
		ReportID string `json:"reportId"`
	}
}

// Create create report for the given RequestCreateReport and return reportId
func (srv *ReportService) Create(req RequestCreateReport) (string, error) {
	log := srv.client.loggerFor("reports").With(
		zap.String("start_date", req.StartDate),
		zap.String("finish_date", req.FinishDate),
		zap.String("sales_channel", string(req.SalesChannel)),
		zap.String("report_type", string(req.TypeReport)),
		zap.Bool("archive", req.IsArchive),
	)
	var res ResponseCreateReport
	log.Info("creating report")
	err := srv.client.httpClient.Post(URL_REPORT_CREATE, req, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to create report", zap.Error(err), zap.String("code", res.Code))
		return res.Result.ReportID, fmt.Errorf("failed to create report: %w", err)
	}
	log.Info("report created", zap.String("report_id", res.Result.ReportID))

	return res.Result.ReportID, nil
}

// ResponseStatusReport is the API response returned when polling report status.
type ResponseStatusReport struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Result      struct {
		ReportID     string       `json:"reportId"`
		Description  string       `json:"description"`
		ReportStatus ReportStatus `json:"reportStatus"`
		PartIds      []string     `json:"partIds"`
	}
}

// Status gets report's status for the given requestId
func (srv *ReportService) Status(requestId string) (ResponseStatusReport, error) {
	log := srv.client.loggerFor("reports").With(zap.String("report_id", requestId))
	var res ResponseStatusReport
	log.Debug("fetching report status")
	err := srv.client.httpClient.Get(fmt.Sprintf(URL_REPORT_STATUS, requestId), &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to fetch report status", zap.Error(err), zap.String("code", res.Code))
		return res, fmt.Errorf("failed to fetch report status: %w", err)
	}
	log.Info("report status fetched",
		zap.String("status", string(res.Result.ReportStatus)),
		zap.Int("parts", len(res.Result.PartIds)),
	)
	return res, nil
}

// Download copies report's data to Writer for the given partId
func (srv *ReportService) Download(partId string, w io.Writer) error {
	log := srv.client.loggerFor("reports").With(zap.String("part_id", partId))
	log.Info("downloading report part")
	err := srv.client.httpClient.Get(fmt.Sprintf(URL_REPORT_DOWNLOAD, partId), w)
	if err != nil {
		log.Error("failed to download report part", zap.Error(err))
		return fmt.Errorf("failed to download report part: %w", err)
	}
	log.Info("report part downloaded")
	return nil
}
