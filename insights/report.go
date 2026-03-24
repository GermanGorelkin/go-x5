package insights

import (
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
)

// ReportService handles communication with the report-related endpoints of the
// X5 Insights API, including trends analysis creation, status polling, and
// export file downloading.
type ReportService service

// ReportResult is a type constraint that enumerates the possible result payloads
// returned by report-related API responses.
type ReportResult interface {
	ResultTrendsAnalysis | ResultReportStatus
}

// ReportResponse is a generic API envelope for report endpoints.
// Code is "ok" on success; Result carries the typed payload.
type ReportResponse[T ReportResult] struct {
	Code   string `json:"code"`
	Result T      `json:"result"`
}

// ----------------------------------------------------------------------------------------------

// ProductSectionID identifies a single node in the product classifier tree.
// Code is the node identifier and Level indicates its depth (e.g. "Ui4").
type ProductSectionID struct {
	Code  string `json:"code"`  // id from ResultTreeProducts
	Level string `json:"level"` // level from ResultTreeProducts
}

// ProductSection wraps a ProductSectionID for inclusion in the report request body.
type ProductSection struct {
	ID ProductSectionID `json:"id"`
}

// Products describes the product selection block of a report request.
// Selection lists the chosen product tree nodes; IsCategoryPluDetailing
// enables PLU-level detail within each category.
type Products struct {
	Selection              []ProductSection `json:"selection"`
	IsCategoryPluDetailing bool             `json:"isCategoryPluDetailing"` // false
}

//----------------------------------------------------------------------------------------------

// ----------------------------------------------------------------------------------------------

// Region represents a single region inside a federal district for the store
// network element list. SelectedFully indicates whether all stores in the
// region are included; CitiesID optionally restricts to specific cities.
type Region struct {
	RegionID      string   `json:"regionId"`
	RegionName    string   `json:"regionName"`
	SelectedFully bool     `json:"selectedFully"`
	CitiesID      []string `json:"citiesId,omitempty"`
}

// FederalDistrict groups regions under a federal district inside a trade
// network element list. SelectedFully indicates whether all stores in the
// district are included.
type FederalDistrict struct {
	DistrictID    int      `json:"districtId"`
	DistrictName  string   `json:"districtName"`
	SelectedFully bool     `json:"selectedFully"`
	Regions       []Region `json:"regions"`
}

// NetworkElementlist describes a single trade network and its geographic
// breakdown used for shop selection in a report request.
type NetworkElementlist struct {
	TradeNetworkID   string            `json:"tradeNetworkId"` // ID from ResultTreeTradeNetworks
	TradeNetworkName string            `json:"tradeNetworkName"`
	SelectedFully    bool              `json:"selectedFully"`
	FederalDistricts []FederalDistrict `json:"federalDistricts"`
}

// Delivery specifies the delivery export strategy for a report request.
// DeliveryMode is one of "EXCLUDE", "INCLUDE_ALL", or "CHOOSE_ONLY_DELIVERY".
// Types lists the selected delivery type IDs (from ResultDeliveryTypes).
type Delivery struct {
	DeliveryMode string   `json:"deliveryMode"` // EXCLUDE|CHOOSE_ONLY_DELIVERY
	Types        []string `json:"types"`        // DeliveryTypeID from ResultDeliveryTypes
}

// SelectedShops aggregates the shop/network selection parameters for a report
// request, including grouping dimensions, growth measure, network elements,
// and delivery configuration.
type SelectedShops struct {
	GroupingAttributes []string             `json:"groupingAttributes"` // TRADE_NETWORK,CITY
	GrowthMeasure      string               `json:"growthMeasure"`      // TOTAL
	NetworkElementlist []NetworkElementlist `json:"networkElementList"`
	// deliveryMode controls export strategy: EXCLUDE omits delivery,
	// CHOOSE_ONLY_DELIVERY includes only delivery types from get_delivery;
	// types is empty for EXCLUDE, lists all delivery IDs for CHOOSE_ONLY_DELIVERY.
	Delivery Delivery `json:"delivery"`
}

//----------------------------------------------------------------------------------------------

// ----------------------------------------------------------------------------------------------

// Customer specifies the customer segmentation for a report.
// CustomerType is typically "TOTAL" to include all customer types.
type Customer struct {
	CustomerType string `json:"customerType"` // TOTAL
}

// Periods describes the time-period configuration for a report, including the
// chosen granularity (e.g. week/month) and the date range.
type Periods struct {
	PeriodGranularityId   string `json:"periodGranularityId"` // id from ResultPeriodGranularity
	PeriodGranularityName string `json:"periodGranularityName"`
	Period                Period `json:"period"`
}

// Period defines a date range with inclusive Start and Stop dates
// formatted as "YYYY-MM-DD".
type Period struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

//----------------------------------------------------------------------------------------------

// RequestTrendsAnalysis is the request body sent to the trends analysis
// report creation endpoint.
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

// PeriodMode selects the time granularity used when building a trends
// analysis request.
type PeriodMode int

const (
	// PeriodMode_Month selects monthly granularity.
	PeriodMode_Month PeriodMode = iota + 1
	// PeriodMode_Week selects weekly granularity.
	PeriodMode_Week
)

// DeliveryMode controls how delivery types are handled in the report export.
type DeliveryMode int

const (
	// DeliveryMode_EXCLUDE omits delivery data from the report.
	DeliveryMode_EXCLUDE DeliveryMode = iota + 1
	// DeliveryMode_CHOOSE_ONLY_DELIVERY includes only selected delivery types.
	DeliveryMode_CHOOSE_ONLY_DELIVERY
	// DeliveryMode_INCLUDE_ALL includes all delivery types in the report.
	DeliveryMode_INCLUDE_ALL
)

// TrendsAnalysisOptions bundles every option needed by BuildRequestTrendsAnalysis
// to construct a complete RequestTrendsAnalysis payload.
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

// BuildRequestTrendsAnalysis assembles a full RequestTrendsAnalysis from the
// given TrendsAnalysisOptions. It resolves product nodes, network elements,
// delivery mode, grouping attributes, period granularity, and date clamping
// against the pre-fetched ReportParameters.
func (srv *ReportService) BuildRequestTrendsAnalysis(opts TrendsAnalysisOptions) (RequestTrendsAnalysis, error) {
	var reqReport RequestTrendsAnalysis
	log := srv.client.loggerFor("reports").With(
		zap.String("report_type", opts.ReportType),
		zap.Time("begin_date", opts.BeginDate),
		zap.Time("end_date", opts.EndDate),
	)
	log.Debug("building trends analysis request")

	// Generate a unique report name from the options (type, period, delivery, dates).
	reqReport.Name = uniqueReportName(opts)
	reqReport.Type = REPORT_TYPE_ID
	reqReport.SectionIDs = opts.Params.SectionIDs()

	// Product selection: iterate over all product IDs from the parameters and
	// wrap each one in a ProductSection for the request payload.
	for _, id := range opts.Params.ProductIDs() {
		reqReport.Parameters.Products.Selection = append(reqReport.Parameters.Products.Selection, ProductSection{
			ID: id,
		})
	}
	reqReport.Parameters.Products.IsCategoryPluDetailing = true

	// Network element list: use the explicitly provided list, or fall back to
	// selecting all trade networks with SelectedFully=true.
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

	// Delivery mode mapping: EXCLUDE omits delivery data (empty types),
	// INCLUDE_ALL keeps all delivery data (empty types),
	// CHOOSE_ONLY_DELIVERY enumerates every delivery ID from the parameters.
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

	// Grouping attributes: use explicitly provided list or default to
	// TOTAL, TRADE_NETWORK, and CITY dimensions.
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

	// Period/granularity selection: resolve the granularity name to its UUID
	// via the fetched parameters dictionary.
	var (
		granularityId   string
		granularityName string
	)
	if opts.PeriodMode == PeriodMode_Month {
		// When selecting "Month" granularity, the period duration must be >= 28 days.
		granularityName = "Месяц"
	} else if opts.PeriodMode == PeriodMode_Week {
		granularityName = "Неделя"
	}
	granularityId = opts.Params.GranularityID(granularityName)

	// Date clamping: ensure the requested end date does not exceed the maximum
	// date available in the system.
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

// uniqueReportName builds a deterministic, human-readable report name by
// combining the report type, period mode, delivery mode, date range, and a
// nanosecond timestamp to guarantee uniqueness.
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

// ResultTrendsAnalysis holds the API response after successfully creating a
// trends analysis report. It contains the report's server-assigned ID,
// type metadata, and audit fields.
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

// CreateTrends sends a RequestTrendsAnalysis to the API and returns the
// created report metadata. The caller should subsequently poll GetReportStatus
// until the export file is ready for download.
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
		return res.Result, fmt.Errorf("failed to create trends report: %w", err)
	}
	log.Info("trends report created", zap.String("report_id", res.Result.ID))
	return res.Result, nil
}

// ----------------------------------------------------------------------------------------------

// Report status lifecycle:
//   CREATED                          - Created, preparing for generation
//   FAILED                           - Error
//   ENQUEUED                         - Queued
//   PROCESSING                       - Being generated
//   EXPORT_FILE_GENERATION_STARTED   - Export file generation started
//   EXPORT_FILE_GENERATED            - Export ready for download

// ResultReportStatus holds the current state of a previously created report,
// including its lifecycle status and the export file ID once generation is
// complete.
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

// GetReportStatus polls the report status endpoint for the given reportID and
// returns the current ResultReportStatus. The caller should check the Status
// field (e.g. "EXPORT_FILE_GENERATED") to decide whether to proceed with
// downloading.
func (srv *ReportService) GetReportStatus(reportID string) (ResultReportStatus, error) {
	log := srv.client.loggerFor("reports").With(zap.String("report_id", reportID))
	url := fmt.Sprintf(URL_REPORT_STATUS, srv.client.API_URL, reportID)
	var res ReportResponse[ResultReportStatus]
	log.Debug("fetching report status")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get report status", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get report status: %w", err)
	}
	log.Info("report status fetched",
		zap.String("status", res.Result.Status),
		zap.String("export_file_id", res.Result.ExportFileID),
	)
	return res.Result, nil
}

// ----------------------------------------------------------------------------------------------

// Download streams the generated export file identified by exportFileID into
// the provided io.Writer. It should be called only after GetReportStatus
// returns Status "EXPORT_FILE_GENERATED".
func (srv *ReportService) Download(exportFileID string, w io.Writer) error {
	log := srv.client.loggerFor("reports").With(zap.String("export_file_id", exportFileID))
	log.Info("downloading report export")
	err := srv.client.httpClient.Get(fmt.Sprintf(URL_REPORT_EXPORT, srv.client.API_URL, exportFileID), w)
	if err != nil {
		log.Error("failed to download report export", zap.Error(err))
		return fmt.Errorf("failed to download report export: %w", err)
	}
	log.Info("report export downloaded")
	return nil
}
