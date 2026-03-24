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
	TS5   SalesChannel = "TS5" // ТС Пятерочка
	TSX   SalesChannel = "TSX" // ТС Перекресток
	TSK   SalesChannel = "TSK" // ТС Карусель
	TSALL SalesChannel = "ALL" // Все каналы

	SALES             TypeReport = "SALES"             // отчет по продажам
	INVENTORY         TypeReport = "INVENTORY"         // отчет по остаткам
	MOVEMENT          TypeReport = "MOVEMENT"          // отчет списания
	CHECK             TypeReport = "CHECK"             // Все каналы
	PRODUCT_DIRECTORY TypeReport = "PRODUCT_DIRECTORY" // Все каналы
	SHOP_DIRECTORY    TypeReport = "SHOP_DIRECTORY"    // Все каналы

	CREATED              ReportStatus = "CREATED"              //создан запрос
	BUILD                ReportStatus = "BUILD"                // отчет в генерации
	DONE                 ReportStatus = "DONE"                 // отчет подготовлен
	ERROR                ReportStatus = "ERROR"                // ошибка при создании отчета
	DOWNLOADED           ReportStatus = "DOWNLOADED"           // отчет загружен
	REMOVAL_EXPIRED_TIME ReportStatus = "REMOVAL_EXPIRED_TIME" // удалено по истечению времени
	REMOVAL_MANUAL       ReportStatus = "REMOVAL_MANUAL"       // удалено администратором вручную
)

type ReportService service

type RequestCreateReport struct {
	FinishDate   string       `json:"finishDate"`
	StartDate    string       `json:"startDate"`
	SalesChannel SalesChannel `json:"salesChannel"`
	TypeReport   TypeReport   `json:"typeReport"`
	IsArchive    bool         `json:"isArchive"`
}
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
