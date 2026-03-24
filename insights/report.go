package insights

import (
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
)

type ReportService service

type ReportResult interface {
	ResultTrendsAnalysis | ResultReportStatus
}

type ReportResponse[T ReportResult] struct {
	Code   string `json:"code"`
	Result T      `json:"result"`
}

// ----------------------------------------------------------------------------------------------
type ProductSectionID struct {
	Code  string `json:"code"`  // id from ResultTreeProducts
	Level string `json:"level"` // level from ResultTreeProducts
}

type ProductSection struct {
	ID ProductSectionID `json:"id"`
}

type Products struct {
	Selection              []ProductSection `json:"selection"`
	IsCategoryPluDetailing bool             `json:"isCategoryPluDetailing"` // false
}

//----------------------------------------------------------------------------------------------

// ----------------------------------------------------------------------------------------------
type Region struct {
	RegionID      string   `json:"regionId"`
	RegionName    string   `json:"regionName"`
	SelectedFully bool     `json:"selectedFully"`
	CitiesID      []string `json:"citiesId,omitempty"`
}

type FederalDistrict struct {
	DistrictID    int      `json:"districtId"`
	DistrictName  string   `json:"districtName"`
	SelectedFully bool     `json:"selectedFully"`
	Regions       []Region `json:"regions"`
}

type NetworkElementlist struct {
	TradeNetworkID   string            `json:"tradeNetworkId"` // ID from ResultTreeTradeNetworks
	TradeNetworkName string            `json:"tradeNetworkName"`
	SelectedFully    bool              `json:"selectedFully"`
	FederalDistricts []FederalDistrict `json:"federalDistricts"`
}

type Delivery struct {
	DeliveryMode string   `json:"deliveryMode"` // EXCLUDE|CHOOSE_ONLY_DELIVERY
	Types        []string `json:"types"`        // DeliveryTypeID from ResultDeliveryTypes
}

type SelectedShops struct {
	GroupingAttributes []string             `json:"groupingAttributes"` // TRADE_NETWORK,CITY
	GrowthMeasure      string               `json:"growthMeasure"`      // TOTAL
	NetworkElementlist []NetworkElementlist `json:"networkElementList"`
	/*
		deliveryMode - будем делать неск выгрузок, одна с параметром EXCLUDE, вторая с параметром CHOOSE_ONLY_DELIVERY
		types - если EXCLUDE, то оставляем пустым, если CHOOSE_ONLY_DELIVERY, то перечисляем все доставки, которые получим в запросе get_delivery
	*/
	Delivery Delivery `json:"delivery"`
}

//----------------------------------------------------------------------------------------------

// ----------------------------------------------------------------------------------------------
type Customer struct {
	CustomerType string `json:"customerType"` // TOTAL
}

type Periods struct {
	PeriodGranularityId   string `json:"periodGranularityId"` // id from ResultPeriodGranularity
	PeriodGranularityName string `json:"periodGranularityName"`
	Period                Period `json:"period"`
}

type Period struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

//----------------------------------------------------------------------------------------------

// RequestTrendsAnalysis - request body for TrendsAnalysis
type RequestTrendsAnalysis struct {
	Name       string   `json:"name"`       // unique name
	Type       string   `json:"type"`       // REPORT_TYPE_ID
	SectionIDs []string `json:"sectionIds"` // id from ResultSections
	Parameters struct {
		Products      Products      `json:"products"`
		SelectedShops SelectedShops `json:"selectedShops"`
		Customers     Customer      `json:"customers"`
		Periods       Periods       `json:"periods"`
		MetricGroups  []string      `json:"metricGroups"` // code from ResultMetricGroups
	} `json:"parameters"`
	Export bool `json:"export"` // true
}

type PeriodMode int

const (
	PeriodMode_Month PeriodMode = iota + 1
	PeriodMode_Week
)

type DeliveryMode int

const (
	DeliveryMode_EXCLUDE DeliveryMode = iota + 1
	DeliveryMode_CHOOSE_ONLY_DELIVERY
	DeliveryMode_INCLUDE_ALL
)

type TrendsAnalysisOptions struct {
	Params             ReportParameters
	PeriodMode         PeriodMode
	DeliveryMode       DeliveryMode
	GroupingAttributes []string // SelectedShops.GroupingAttributes
	BeginDate          time.Time
	EndDate            time.Time
	NetworkElementlist []NetworkElementlist

	ReportType string // TRENDS_ANALYSIS_DATA|TRENDS_ANALYSIS_WD|TRENDS_ANALYSIS_REGION
}

func (srv *ReportService) BuildRequestTrendsAnalysis(opts TrendsAnalysisOptions) (RequestTrendsAnalysis, error) {
	var reqReport RequestTrendsAnalysis
	log := srv.client.loggerFor("reports").With(
		zap.String("report_type", opts.ReportType),
		zap.Time("begin_date", opts.BeginDate),
		zap.Time("end_date", opts.EndDate),
	)
	log.Debug("building trends analysis request")

	reqReport.Name = uniqueReportName(opts)
	reqReport.Type = REPORT_TYPE_ID
	reqReport.SectionIDs = opts.Params.SectionIDs()

	for _, id := range opts.Params.ProductIDs() {
		reqReport.Parameters.Products.Selection = append(reqReport.Parameters.Products.Selection, ProductSection{
			ID: id,
		})
	}
	reqReport.Parameters.Products.IsCategoryPluDetailing = true

	networks := opts.NetworkElementlist
	if len(networks) == 0 {
		for _, id := range opts.Params.TradeNetworkIDs() {
			nel := NetworkElementlist{
				TradeNetworkID:   id,
				SelectedFully:    true,
				FederalDistricts: []FederalDistrict{},
			}
			networks = append(networks, nel)
		}
	}

	var delivery Delivery
	if opts.DeliveryMode == DeliveryMode_EXCLUDE {
		delivery = Delivery{
			DeliveryMode: "EXCLUDE",
			Types:        []string{},
		}
	} else if opts.DeliveryMode == DeliveryMode_INCLUDE_ALL {
		delivery = Delivery{
			DeliveryMode: "INCLUDE_ALL",
			Types:        []string{},
		}
	} else {
		delivery = Delivery{
			DeliveryMode: "CHOOSE_ONLY_DELIVERY",
			Types:        opts.Params.DeliveryIDs(),
		}
	}

	var groupingAttributes []string
	if len(opts.GroupingAttributes) > 0 {
		groupingAttributes = append(groupingAttributes, opts.GroupingAttributes...)
	} else {
		groupingAttributes = []string{"TOTAL", "TRADE_NETWORK", "CITY"}
	}

	reqReport.Parameters.SelectedShops = SelectedShops{
		GroupingAttributes: groupingAttributes,
		GrowthMeasure:      "TOTAL",
		NetworkElementlist: networks,
		Delivery:           delivery,
	}

	reqReport.Parameters.Customers = Customer{
		CustomerType: "TOTAL",
	}

	var (
		granularityId   string
		granularityName string
	)
	if opts.PeriodMode == PeriodMode_Month {
		// "При выборе гранулярности 'Месяц' продолжительность периода должна быть больше или равна 28 дням"
		granularityName = "Месяц"
	} else if opts.PeriodMode == PeriodMode_Week {
		granularityName = "Неделя"
	}
	granularityId = opts.Params.GranularityID(granularityName)

	_, maxDR, err := opts.Params.AvailableDates()
	if err != nil {
		log.Error("failed to parse available dates", zap.Error(err))
		return reqReport, fmt.Errorf("failed to get available dates: %w", err)
	}
	if opts.EndDate.After(maxDR) {
		log.Debug("end date exceeds max available date", zap.Time("max_available_date", maxDR))
		opts.EndDate = maxDR
	}

	reqReport.Parameters.Periods = Periods{
		PeriodGranularityId:   granularityId,
		PeriodGranularityName: granularityName,
		Period: Period{
			Start: opts.BeginDate.Format("2006-01-02"),
			Stop:  opts.EndDate.Format("2006-01-02"),
		},
	}

	reqReport.Parameters.MetricGroups = opts.Params.MetricIDs()

	reqReport.Export = true
	log.Info("trends analysis request built",
		zap.String("request_name", reqReport.Name),
		zap.Int("section_ids", len(reqReport.SectionIDs)),
		zap.Int("product_nodes", len(reqReport.Parameters.Products.Selection)),
		zap.Int("metric_groups", len(reqReport.Parameters.MetricGroups)),
		zap.String("granularity", reqReport.Parameters.Periods.PeriodGranularityName),
		zap.String("delivery_mode", reqReport.Parameters.SelectedShops.Delivery.DeliveryMode),
	)

	return reqReport, nil
}

func uniqueReportName(opts TrendsAnalysisOptions) string {
	var deliveryMode string
	if opts.DeliveryMode == DeliveryMode_EXCLUDE {
		deliveryMode = "EXCLUDE"
	} else if opts.DeliveryMode == DeliveryMode_INCLUDE_ALL {
		deliveryMode = "INCLUDE_ALL"
	} else {
		deliveryMode = "CHOOSE_ONLY_DELIVERY"
	}

	var periodMode string
	if opts.PeriodMode == PeriodMode_Month {
		periodMode = "MONTH"
	} else if opts.PeriodMode == PeriodMode_Week {
		periodMode = "WEEK"
	}

	beginDate := opts.BeginDate.Format("20060102")
	endDate := opts.EndDate.Format("20060102")

	return fmt.Sprintf("%s-%s-%s-%s-%s-%d", opts.ReportType, periodMode, deliveryMode, beginDate, endDate, time.Now().UnixNano())
}

// ResultTrendsAnalysis - response body for TrendsAnalysis
type ResultTrendsAnalysis struct {
	ID             string `json:"id"`
	ReportTypeID   string `json:"reportTypeId"`
	ReportTypeName string `json:"reportTypeName"`
	ReportName     string `json:"reportName"`
	ReportStatusid string `json:"reportStatusId"`
	CreatedAt      string `json:"createdAt"`
	CreatedBy      string `json:"createdBy"`
	AccountID      string `json:"accountId"`
	ParametersID   string `json:"parametersId"`
}

// CreateTrends creates trends analysis
func (srv *ReportService) CreateTrends(request RequestTrendsAnalysis) (ResultTrendsAnalysis, error) {
	log := srv.client.loggerFor("reports").With(
		zap.String("request_name", request.Name),
		zap.String("report_type_id", request.Type),
	)
	url := fmt.Sprintf(URL_CREATE_TRENDS, srv.client.API_URL)
	var res ReportResponse[ResultTrendsAnalysis]
	log.Info("creating trends report")
	err := srv.client.httpClient.Post(url, request, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to create trends report", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to create trends: %v", err)
	}
	log.Info("trends report created", zap.String("report_id", res.Result.ID))
	return res.Result, nil
}

// ----------------------------------------------------------------------------------------------

/*
CREATED- Создан, готовится к генерации
FAILED - Ошибка
ENQUEUED - В очереди
PROCESSING - Формируется
EXPORT_FILE_GENERATION_STARTED - Начата генерация файла для выгрузки
EXPORT_FILE_GENERATED - Выгрузка готова к скачиванию
*/

type ResultReportStatus struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Deleted      bool   `json:"deleted"`
	CreatedAt    string `json:"createdAt"`
	CreatedBy    string `json:"createdBy"`
	AccountID    string `json:"accountId"`
	ParametersID string `json:"parametersId"`
	ExportFileID string `json:"exportFileId"`
}

func (srv *ReportService) GetReportStatus(reportID string) (ResultReportStatus, error) {
	log := srv.client.loggerFor("reports").With(zap.String("report_id", reportID))
	url := fmt.Sprintf(URL_REPORT_STATUS, srv.client.API_URL, reportID)
	var res ReportResponse[ResultReportStatus]
	log.Debug("fetching report status")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get report status", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get report status: %v", err)
	}
	log.Info("report status fetched",
		zap.String("status", res.Result.Status),
		zap.String("export_file_id", res.Result.ExportFileID),
	)
	return res.Result, nil
}

// ----------------------------------------------------------------------------------------------

func (srv *ReportService) Download(exportFileID string, w io.Writer) error {
	log := srv.client.loggerFor("reports").With(zap.String("export_file_id", exportFileID))
	log.Info("downloading report export")
	err := srv.client.httpClient.Get(fmt.Sprintf(URL_REPORT_EXPORT, srv.client.API_URL, exportFileID), w)
	if err != nil {
		log.Error("failed to download report export", zap.Error(err))
		return fmt.Errorf("failed to download:%w", err)
	}
	log.Info("report export downloaded")
	return nil
}
