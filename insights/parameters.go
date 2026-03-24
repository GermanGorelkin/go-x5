package insights

import (
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
)

// REPORT_TYPE_ID is the hardcoded UUID that identifies the TrendsAnalysis report type
// across all parameter and report API calls.
const (
	REPORT_TYPE_ID = "8ddb5b9f-2193-453c-96ba-a0a3c14e517c" //
)

// ParametersService handles fetching report configuration parameters (sections, dates, stores, products, etc.).
type ParametersService service

// ParametersResult is a type constraint listing every concrete result type that can appear
// inside a generic ParametersResponse.
type ParametersResult interface {
	ResultSections | ResultAvailableDates | ResultTreeStores | ResultTreeProducts | ResultDelivery | ResultMetrics | ResultGranularities
}

// ParametersResponse is a generic JSON envelope returned by every parameters endpoint.
// Code is "ok" on success; Result carries the typed payload.
type ParametersResponse[T ParametersResult] struct {
	Code   string `json:"code"`
	Result T      `json:"result"`
}

// ----------------------------------------------------------------------------------------------

// ReportParameters aggregates all dictionary data needed to build a report request.
// It embeds individual result types fetched from separate API endpoints so that
// helper methods can derive IDs, dates, and trees from a single value.
type ReportParameters struct {
	ResultSections
	ResultAvailableDates
	ResultTreeStores
	ResultTreeProducts
	ResultDelivery
	ResultMetrics
	ResultGranularities
}

// SectionIDs returns the IDs of report sections that belong to the TrendsAnalysis report type.
func (rp ReportParameters) SectionIDs() []string {
	var ids []string
	for _, section := range rp.Reportsections {
		if section.ReportTypeID == REPORT_TYPE_ID {
			ids = append(ids, section.ID)
		}
	}
	return ids
}

// AvailableDates parses the min/max date strings from the API into time.Time values.
// These boundaries define the allowed date range for report period selection.
func (rp ReportParameters) AvailableDates() (minDT time.Time, maxDT time.Time, err error) {
	minDT, err = time.Parse("2006-01-02", rp.MinDT)
	if err != nil {
		return minDT, maxDT, fmt.Errorf("failed to parse minDT(%s): %v", rp.MaxDT, err)
	}
	maxDT, err = time.Parse("2006-01-02", rp.MaxDT)
	if err != nil {
		return minDT, maxDT, fmt.Errorf("failed to parse maxDT(%s): %v", rp.MaxDT, err)
	}

	return minDT, maxDT, nil
}

// TradeNetworkIDs extracts the unique trade-network identifiers from the store classifier tree.
func (rp ReportParameters) TradeNetworkIDs() []string {
	var ids []string
	for _, network := range rp.ResultTreeStores.TradeNetworks {
		ids = append(ids, network.ID)
	}
	return ids
}

// FederalDistricts flattens the store classifier tree into a slice of FederalDistrict values,
// one per (district, region) pair, suitable for inclusion in a report request.
// Duplicate (districtID, regionID) pairs are deduplicated.
func (rp ReportParameters) FederalDistricts() []FederalDistrict {
	var federals []FederalDistrict

	for _, network := range rp.ResultTreeStores.TradeNetworks {
		for _, federalDist := range network.FederalDistricts {
			for _, region := range federalDist.Regions {
				r := Region{
					RegionID:      region.ID,
					RegionName:    region.Name,
					SelectedFully: true,
					CitiesID:      []string{},
				}
				f := FederalDistrict{
					DistrictID:    federalDist.ID,
					DistrictName:  federalDist.Name,
					SelectedFully: false,
					Regions:       []Region{r},
				}

				// check for duplicates
				found := false
				for i := range federals {
					if federals[i].DistrictID == f.DistrictID && federals[i].Regions[0].RegionID == r.RegionID {
						found = true
						break
					}
				}
				if found {
					continue
				}
				//

				federals = append(federals, f)
			}
		}
	}

	return federals
}

/*
func (rp ReportParameters) FederalDistrictsWithCities() []FederalDistrict {
	var federals []FederalDistrict

	f, err := os.Create("FederalDistrict.txt")
	if err != nil {
		log.Panicf("%s", err)
	}
	defer f.Close()

	for _, network := range rp.ResultTreeStores.TradeNetworks {
		for _, federalDist := range network.FederalDistricts {
			for _, region := range federalDist.Regions {
				for _, city := range region.Cities {
					fmt.Fprintf(f, "%s;%s;%d;%s;%s;%s;%s;%s\n", network.ID, network.Name, federalDist.ID, federalDist.Name, region.ID, region.Name, city.ID, city.Name)

					// log.Printf("%s;%s;%d;%s;%s;%s;%s", network.ID, network.Name, federalDist.ID, federalDist.Name, region.ID, city.ID, city.Name)

					// r := Region{
					// 	RegionID:      region.ID,
					// 	RegionName:    region.Name,
					// 	SelectedFully: true,
					// 	CitiesID:      []string{},
					// }
					// f := FederalDistrict{
					// 	DistrictID:    federalDist.ID,
					// 	DistrictName:  federalDist.Name,
					// 	SelectedFully: false,
					// 	Regions:       []Region{r},
					// }

					// // check for duplicates
					// found := false
					// for i := range federals {
					// 	if federals[i].DistrictID == f.DistrictID && federals[i].Regions[0].RegionID == r.RegionID {
					// 		found = true
					// 		break
					// 	}
					// }
					// if found {
					// 	continue
					// }
					// //

					// federals = append(federals, f)
				}
			}
		}
	}

	return federals
}
*/

// ProductIDs collects product identifiers at the 4th (deepest) level of the product tree.
// These correspond to the most granular product categories (level "Ui4").
func (rp ReportParameters) ProductIDs() []ProductSectionID {
	var ids []ProductSectionID
	for _, node1 := range rp.ResultTreeProducts.Nodes {
		for _, node2 := range node1.Children {
			for _, node3 := range node2.Children {
				for _, node3 := range node3.Children {
					ids = append(ids, ProductSectionID{node3.ID, node3.Level}) // "level": "Ui4"
				}
			}
		}
	}
	return ids
}

// GetAllProductIDs collects product identifiers from every level of the product tree,
// not just the leaf nodes. This is useful for full-catalog exports.
func (rp ReportParameters) GetAllProductIDs() []ProductSectionID {
	var ids []ProductSectionID
	for _, node := range rp.ResultTreeProducts.Nodes {
		ids = append(ids, ProductSectionID{node.ID, node.Level})
		ids = append(ids, getProductIDs(node)...)
	}
	return ids
}

// getProductIDs recursively walks the children of a TreeProductNodes and collects their IDs.
func getProductIDs(nodes TreeProductNodes) []ProductSectionID {
	var ids []ProductSectionID
	for _, node := range nodes.Children {
		ids = append(ids, ProductSectionID{node.ID, node.Level})
		if len(node.Children) > 0 {
			ids = append(ids, getProductIDs(node)...)
		}
	}
	return ids
}

// DeliveryIDs extracts all delivery-type identifiers from the delivery dictionary.
func (rp ReportParameters) DeliveryIDs() []string {
	var ids []string
	for _, delivery := range rp.ResultDelivery.Types {
		ids = append(ids, delivery.DeliveryTypeID)
	}
	return ids
}

// MetricIDs extracts all metric-group codes from the metrics dictionary.
func (rp ReportParameters) MetricIDs() []string {
	var ids []string
	for _, metric := range rp.ResultMetrics.MetricGroups {
		ids = append(ids, metric.Code)
	}
	return ids
}

// GranularityID looks up a period granularity by its display name (e.g. "Месяц", "Неделя")
// and returns the corresponding API identifier. Returns an empty string if not found.
func (rp ReportParameters) GranularityID(name string) string {
	for _, granularity := range rp.ResultGranularities.Granularities {
		if granularity.Name == name {
			return granularity.ID
		}
	}
	return ""
}

//-----------------

// FetchReportParameters calls every parameter endpoint in sequence and returns
// a fully populated ReportParameters value containing sections, dates, store/product trees,
// delivery types, metrics, and granularities.
func (srv *ParametersService) FetchReportParameters() (ReportParameters, error) {
	var parameters ReportParameters
	log := srv.client.loggerFor("parameters")
	log.Info("fetching report parameters")

	sections, err := srv.GetSections()
	if err != nil {
		log.Error("failed to fetch report sections", zap.Error(err))
		return parameters, fmt.Errorf("failed to fetch report sections: %w", err)
	}
	parameters.ResultSections = sections

	dates, err := srv.GetAvailableDates()
	if err != nil {
		log.Error("failed to fetch available dates", zap.Error(err))
		return parameters, fmt.Errorf("failed to fetch available dates: %w", err)
	}
	parameters.ResultAvailableDates = dates

	stores, err := srv.GetTreeStores()
	if err != nil {
		log.Error("failed to fetch store tree", zap.Error(err))
		return parameters, fmt.Errorf("failed to fetch store tree: %w", err)
	}
	parameters.ResultTreeStores = stores

	products, err := srv.GetTreeProducts()
	if err != nil {
		log.Error("failed to fetch product tree", zap.Error(err))
		return parameters, fmt.Errorf("failed to fetch product tree: %w", err)
	}
	parameters.ResultTreeProducts = products

	delivery, err := srv.GetDelivery()
	if err != nil {
		log.Error("failed to fetch delivery dictionary", zap.Error(err))
		return parameters, fmt.Errorf("failed to fetch delivery dictionary: %w", err)
	}
	parameters.ResultDelivery = delivery

	granularities, err := srv.GetGranularities()
	if err != nil {
		log.Error("failed to fetch granularities", zap.Error(err))
		return parameters, fmt.Errorf("failed to fetch granularities: %w", err)
	}
	parameters.ResultGranularities = granularities

	metrics, err := srv.GetMetrics()
	if err != nil {
		log.Error("failed to fetch metrics", zap.Error(err))
		return parameters, fmt.Errorf("failed to fetch metrics: %w", err)
	}
	parameters.ResultMetrics = metrics
	log.Info("report parameters fetched",
		zap.Int("sections", len(parameters.ResultSections.Reportsections)),
		zap.Int("trade_networks", len(parameters.ResultTreeStores.TradeNetworks)),
		zap.Int("product_roots", len(parameters.ResultTreeProducts.Nodes)),
		zap.Int("delivery_types", len(parameters.ResultDelivery.Types)),
		zap.Int("metrics", len(parameters.ResultMetrics.MetricGroups)),
		zap.Int("granularities", len(parameters.ResultGranularities.Granularities)),
	)

	return parameters, nil
}

//----------------------------------------------------------------------------------------------
// Report section list

// ResultSections holds the list of report sections returned by the build-sections endpoint.
type ResultSections struct {
	Reportsections []struct {
		ID           string `json:"id"`
		ReportTypeID string `json:"reportTypeId"`
		Name         string `json:"name"`
	} `json:"reportSections"`
}

// GetSections returns all report sections for REPORT_TYPE_ID
func (srv *ParametersService) GetSections() (ResultSections, error) {
	log := srv.client.loggerFor("parameters")
	url := fmt.Sprintf(URL_BUILD_SECTIONS, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultSections]
	log.Debug("fetching report sections")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get report sections", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get report sections: %w", err)
	}
	log.Debug("report sections fetched", zap.Int("count", len(res.Result.Reportsections)))
	return res.Result, nil
}

//----------------------------------------------------------------------------------------------
// Available dates for report building

// ResultAvailableDates holds the minimum and maximum dates that can be used for report periods.
type ResultAvailableDates struct {
	MinDT string `json:"minDt"`
	MaxDT string `json:"maxDt"`
}

// GetAvailableDates returns all available dates for REPORT_TYPE_ID
func (srv *ParametersService) GetAvailableDates() (ResultAvailableDates, error) {
	log := srv.client.loggerFor("parameters")
	url := fmt.Sprintf(URL_BUILD_AVAILABLE_DATE, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultAvailableDates]
	log.Debug("fetching available dates")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get available dates", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get available dates: %w", err)
	}
	log.Debug("available dates fetched",
		zap.String("min_dt", res.Result.MinDT),
		zap.String("max_dt", res.Result.MaxDT),
	)
	return res.Result, nil
}

//----------------------------------------------------------------------------------------------
// Store classifier tree

// ResultTreeStores represents the hierarchical store classifier returned by the tree/stores endpoint.
// The tree is organised as TradeNetworks → FederalDistricts → Regions → Cities.
type ResultTreeStores struct {
	TotalStores   int `json:"totalStores"`
	TradeNetworks []struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		Storescount      int    `json:"storesCount"`
		FederalDistricts []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			StoresCount int    `json:"storesCount"`
			Regions     []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				StoresCount int    `json:"storesCount"`
				Cities      []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					StoresCount int    `json:"storesCount"`
				} `json:"cities"`
			} `json:"regions"`
		} `json:"federalDistricts"`
	} `json:"tradeNetworks"`
}

// GetTreeStores gets the tree stores for REPORT_TYPE_ID
func (srv *ParametersService) GetTreeStores() (ResultTreeStores, error) {
	log := srv.client.loggerFor("parameters")
	url := fmt.Sprintf(URL_TREE_STORES, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultTreeStores]
	log.Debug("fetching store tree")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get store tree", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get store tree: %w", err)
	}
	log.Debug("store tree fetched",
		zap.Int("trade_networks", len(res.Result.TradeNetworks)),
		zap.Int("total_stores", res.Result.TotalStores),
	)
	return res.Result, nil
}

//----------------------------------------------------------------------------------------------
// Product classifier tree

// TreeProductNodes is a recursive node in the product classification tree.
// Each node has an ID, a human-readable Name, a Level indicator (e.g. "Ui1"…"Ui4"),
// and zero or more Children.
type TreeProductNodes struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Level    string             `json:"level"`
	Children []TreeProductNodes `json:"children"`
}

// ResultTreeProducts holds the root nodes of the product classifier tree.
type ResultTreeProducts struct {
	Nodes []TreeProductNodes `json:"nodes"`
}

// GetTreeProducts gets the tree products for REPORT_TYPE_ID
func (srv *ParametersService) GetTreeProducts() (ResultTreeProducts, error) {
	log := srv.client.loggerFor("parameters")
	url := fmt.Sprintf(URL_TREE_PRODUCTS, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultTreeProducts]
	log.Debug("fetching product tree")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get product tree", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get product tree: %w", err)
	}
	log.Debug("product tree fetched", zap.Int("roots", len(res.Result.Nodes)))
	return res.Result, nil
}

// Delivery types list

// ResultDelivery holds the available delivery types returned by the delivery dictionary endpoint.
type ResultDelivery struct {
	Types []struct {
		DeliveryTypeID   string `json:"deliveryTypeId"`
		DeliveryTypeName string `json:"deliveryTypeName"`
		Icon             string `json:"icon"`
		DateStart        string `json:"dateStart"`
	} `json:"types"`
}

// GetDelivery gets the list of delivery
func (srv *ParametersService) GetDelivery() (ResultDelivery, error) {
	log := srv.client.loggerFor("parameters")
	url := fmt.Sprintf(URL_DELIVERY, srv.client.API_URL)
	var res ParametersResponse[ResultDelivery]
	log.Debug("fetching delivery types")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get delivery types", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get delivery types: %w", err)
	}
	log.Debug("delivery types fetched", zap.Int("count", len(res.Result.Types)))
	return res.Result, nil
}

// Metrics list

// ResultMetrics holds the available metric groups returned by the metrics endpoint.
// Each group contains a Code identifier and a list of individual Metrics.
type ResultMetrics struct {
	MetricGroups []struct {
		Code    string   `json:"code"`
		Metrics []string `json:"metrics"`
	} `json:"metricGroups"`
}

// GetMetrics gets the list of metrics
func (srv *ParametersService) GetMetrics() (ResultMetrics, error) {
	log := srv.client.loggerFor("parameters")
	url := fmt.Sprintf(URL_METRICS, srv.client.API_URL)
	var res ParametersResponse[ResultMetrics]
	log.Debug("fetching metrics")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get metrics", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get metrics: %w", err)
	}
	log.Debug("metrics fetched", zap.Int("count", len(res.Result.MetricGroups)))
	return res.Result, nil
}

// Granularities

// ResultGranularities holds the available period granularities (e.g. week, month)
// returned by the periods dictionary endpoint.
type ResultGranularities struct {
	Granularities []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"granularities"`
}

// GetGranularities gets the list of granularities
func (srv *ParametersService) GetGranularities() (ResultGranularities, error) {
	log := srv.client.loggerFor("parameters")
	url := fmt.Sprintf(URL_GRANULARITIES, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultGranularities]
	log.Debug("fetching granularities")
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		log.Error("failed to get granularities", zap.Error(err), zap.String("code", res.Code))
		return res.Result, fmt.Errorf("failed to get granularities: %w", err)
	}
	log.Debug("granularities fetched", zap.Int("count", len(res.Result.Granularities)))
	return res.Result, nil
}

// Products

// RequestProductsDownload is the request body sent to the products/download endpoint
// to export a subset of the product tree as a file.
type RequestProductsDownload struct {
	Nodes         []RequestProductsDownloadNode `json:"nodes"`
	GlobalCatalog bool                          `json:"global_catalog"`
}

// RequestProductsDownloadNode identifies a single product tree node to include in the export.
type RequestProductsDownloadNode struct {
	ID    string `json:"id"`
	Level string `json:"level"`
}

// ConvertToRequestProductsDownloadNode converts a slice of ProductSectionID values
// (used internally for report building) into the download-specific node format.
func ConvertToRequestProductsDownloadNode(src []ProductSectionID) []RequestProductsDownloadNode {
	result := make([]RequestProductsDownloadNode, len(src))
	for i := range src {
		result[i] = RequestProductsDownloadNode{
			ID:    src[i].Code,
			Level: src[i].Level,
		}
	}
	return result
}

// ProductsDownload sends a POST request to export selected product tree nodes
// and streams the resulting file into w.
func (srv *ParametersService) ProductsDownload(rpd RequestProductsDownload, w io.Writer) error {
	log := srv.client.loggerFor("parameters").With(zap.Int("nodes", len(rpd.Nodes)))
	log.Info("downloading products export")
	err := srv.client.httpClient.Post(fmt.Sprintf(URL_PRODUCTS_EXPORT, srv.client.API_URL), rpd, w)
	if err != nil {
		log.Error("failed to download products export", zap.Error(err))
		return fmt.Errorf("failed to download products export: %w", err)
	}
	log.Info("products export downloaded")
	return nil
}
