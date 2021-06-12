package logistics

import "fmt"

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

	CREATED              ReportStatus = "ReportStatus"         //создан запрос
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
	Code        string
	Description string
	Result      struct {
		ReportID string `json:"reportId"`
	}
}

// Create create report for the given RequestCreateReport and return list of reportId
func (srv *ReportService) Create(req RequestCreateReport) (string, error) {
	var res ResponseCreateReport
	err := srv.client.httpClient.Post(URL_REPORT_CREATE, req, &res)
	if err != nil || res.Code != "ok" {
		return res.Result.ReportID, fmt.Errorf("failed to create report:%w", err)
	}

	return res.Result.ReportID, nil
}
