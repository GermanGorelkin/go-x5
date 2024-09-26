package insights

import (
	"fmt"
	"io"
	"time"
)

const (
	REPORT_TYPE_ID = "8ddb5b9f-2193-453c-96ba-a0a3c14e517c" //
)

type ParametersService service

type ParametersResult interface {
	ResultSections | ResultAvailableDates | ResultTreeStores | ResultTreeProducts | ResultDelivery | ResultMetrics | ResultGranularities
}
type ParametersResponse[T ParametersResult] struct {
	Code   string `json:"code"`
	Result T      `json:"result"`
}

// ----------------------------------------------------------------------------------------------

type ReportParameters struct {
	ResultSections
	ResultAvailableDates
	ResultTreeStores
	ResultTreeProducts
	ResultDelivery
	ResultMetrics
	ResultGranularities
}

func (rp ReportParameters) SectionIDs() []string {
	var ids []string
	for _, section := range rp.Reportsections {
		if section.ReportTypeID == REPORT_TYPE_ID {
			ids = append(ids, section.ID)
		}
	}
	return ids
}

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

func (rp ReportParameters) TradeNetworkIDs() []string {
	var ids []string
	for _, network := range rp.ResultTreeStores.TradeNetworks {
		ids = append(ids, network.ID)
	}
	return ids
}

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

				// ищем дубли
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

					// // ищем дубли
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

// ProductIDs gets ids of products 4 lvl
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

// ProductIDs gets ids of products all lvl
func (rp ReportParameters) GetAllProductIDs() []ProductSectionID {
	var ids []ProductSectionID
	for _, node := range rp.ResultTreeProducts.Nodes {
		ids = append(ids, ProductSectionID{node.ID, node.Level})
		ids = append(ids, getProductIDs(node)...)
	}
	return ids
}

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

func (rp ReportParameters) DeliveryIDs() []string {
	var ids []string
	for _, delivery := range rp.ResultDelivery.Types {
		ids = append(ids, delivery.DeliveryTypeID)
	}
	return ids
}

func (rp ReportParameters) MetricIDs() []string {
	var ids []string
	for _, metric := range rp.ResultMetrics.MetricGroups {
		ids = append(ids, metric.Code)
	}
	return ids
}

func (rp ReportParameters) GranularityID(name string) string {
	for _, granularity := range rp.ResultGranularities.Granularities {
		if granularity.Name == name {
			return granularity.ID
		}
	}
	return ""
}

//-----------------

func (srv *ParametersService) FetchReportParameters() (ReportParameters, error) {
	var parameters ReportParameters

	sections, err := srv.GetSections()
	if err != nil {
		return parameters, fmt.Errorf("failed to GetSections:%w", err)
	}
	parameters.ResultSections = sections

	dates, err := srv.GetAvailableDates()
	if err != nil {
		return parameters, fmt.Errorf("failed to GetAvailableDates:%w", err)
	}
	parameters.ResultAvailableDates = dates

	stores, err := srv.GetTreeStores()
	if err != nil {
		return parameters, fmt.Errorf("failed to GetTreeStores:%w", err)
	}
	parameters.ResultTreeStores = stores

	products, err := srv.GetTreeProducts()
	if err != nil {
		return parameters, fmt.Errorf("failed to GetTreeProducts:%w", err)
	}
	parameters.ResultTreeProducts = products

	delivery, err := srv.GetDelivery()
	if err != nil {
		return parameters, fmt.Errorf("failed to GetDelivery:%w", err)
	}
	parameters.ResultDelivery = delivery

	granularities, err := srv.GetGranularities()
	if err != nil {
		return parameters, fmt.Errorf("failed to GetGranularities:%w", err)
	}
	parameters.ResultGranularities = granularities

	metrics, err := srv.GetMetrics()
	if err != nil {
		return parameters, fmt.Errorf("failed to GetMetrics:%w", err)
	}
	parameters.ResultMetrics = metrics

	return parameters, nil
}

//----------------------------------------------------------------------------------------------
// Список блоков для отчета

// ResultSections
type ResultSections struct {
	Reportsections []struct {
		ID           string `json:"id"`
		ReportTypeID string `json:"reportTypeId"`
		Name         string `json:"name"`
	} `json:"reportSections"`
}

// GetSections returns all report sections for REPORT_TYPE_ID
func (srv *ParametersService) GetSections() (ResultSections, error) {
	url := fmt.Sprintf(URL_BUILD_SECTIONS, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultSections]
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to get sections: %v", err)
	}
	return res.Result, nil
}

//----------------------------------------------------------------------------------------------
// Доступные даты для построения отчета

// ResultAvailableDates
type ResultAvailableDates struct {
	MinDT string `json:"minDt"`
	MaxDT string `json:"maxDt"`
}

// GetAvailableDates returns all available dates for REPORT_TYPE_ID
func (srv *ParametersService) GetAvailableDates() (ResultAvailableDates, error) {
	url := fmt.Sprintf(URL_BUILD_AVAILABLE_DATE, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultAvailableDates]
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to get available dates: %v", err)
	}
	return res.Result, nil
}

//----------------------------------------------------------------------------------------------
// Дерево-классификатор магазинов

// ResultTreeStores
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
	url := fmt.Sprintf(URL_TREE_STORES, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultTreeStores]
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to get tree stores: %v", err)
	}
	return res.Result, nil
}

//----------------------------------------------------------------------------------------------
// Дерево-классификатор товаров

type TreeProductNodes struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Level    string             `json:"level"`
	Children []TreeProductNodes `json:"children"`
}

// ResultTreeProducts
type ResultTreeProducts struct {
	Nodes []TreeProductNodes `json:"nodes"`
}

// GetTreeProducts gets the tree products for REPORT_TYPE_ID
func (srv *ParametersService) GetTreeProducts() (ResultTreeProducts, error) {
	url := fmt.Sprintf(URL_TREE_PRODUCTS, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultTreeProducts]
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to get tree products: %v", err)
	}
	return res.Result, nil
}

// Список доставок

// ResultDelivery
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
	url := fmt.Sprintf(URL_DELIVERY, srv.client.API_URL)
	var res ParametersResponse[ResultDelivery]
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to get delivery: %v", err)
	}
	return res.Result, nil
}

// Список метрик

// ResultMetrics
type ResultMetrics struct {
	MetricGroups []struct {
		Code    string   `json:"code"`
		Metrics []string `json:"metrics"`
	} `json:"metricGroups"`
}

// GetMetrics gets the list of metrics
func (srv *ParametersService) GetMetrics() (ResultMetrics, error) {
	url := fmt.Sprintf(URL_METRICS, srv.client.API_URL)
	var res ParametersResponse[ResultMetrics]
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to get metrics: %v", err)
	}
	return res.Result, nil
}

// Granularities

// ResultGranularities
type ResultGranularities struct {
	Granularities []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"granularities"`
}

// GetGranularities gets the list of granularities
func (srv *ParametersService) GetGranularities() (ResultGranularities, error) {
	url := fmt.Sprintf(URL_GRANULARITIES, srv.client.API_URL, REPORT_TYPE_ID)
	var res ParametersResponse[ResultGranularities]
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res.Result, fmt.Errorf("failed to get granularities: %v", err)
	}
	return res.Result, nil
}

// Products

type RequestProductsDownload struct {
	Nodes         []RequestProductsDownloadNode `json:"nodes"`
	GlobalCatalog bool                          `json:"global_catalog"`
}

type RequestProductsDownloadNode struct {
	ID    string `json:"id"`
	Level string `json:"level"`
}

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

func (srv *ParametersService) ProductsDownload(rpd RequestProductsDownload, w io.Writer) error {
	err := srv.client.httpClient.Post(fmt.Sprintf(URL_PRODUCTS_EXPORT, srv.client.API_URL), rpd, w)
	if err != nil {
		return fmt.Errorf("failed to download:%w", err)
	}
	return nil
}
