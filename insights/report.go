package insights

import "fmt"

type ReportService service

type ReportResult interface {
	ResultTrendsAnalysis
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
	SelectedFully bool     `json:"selectedFully"`
	CitiesID      []string `json:"citiesId,omitempty"`
}

type FederalDistrict struct {
	DistrictID    int      `json:"districtId"`
	SelectedFully bool     `json:"selectedFully"`
	Regions       []Region `json:"regions"`
}

type NetworkElementlist struct {
	TradeNetworkID   string            `json:"tradeNetworkId"` // ID from ResultTreeTradeNetworks
	SelectedFully    bool              `json:"selectedFully"`  // true
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
	PeriodGranularityId string `json:"periodGranularityId"` // id from ResultPeriodGranularity
	Period              Period `json:"period"`
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
	url := fmt.Sprintf(URL_CREATE_TRENDS, srv.client.API_URL)
	var res ReportResponse[ResultTrendsAnalysis]
	err := srv.client.httpClient.Post(url, request, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to create trends: %v", err)
	}
	return res.Result, nil
}
