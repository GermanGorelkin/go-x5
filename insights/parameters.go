package insights

import "fmt"

const (
	REPORT_TYPE_ID = "8ddb5b9f-2193-453c-96ba-a0a3c14e517c" //
)

type ParametersService service

type ParametersResult interface {
	ResultSections | ResultAvailableDates | ResultTreeStores | ResultTreeProducts
}
type ParametersResponse[T ParametersResult] struct {
	Code   string `json:"code"`
	Result T      `json:"result"`
}

//----------------------------------------------------------------------------------------------
// Список блоков для отчета

// ResultSections
type ResultSections struct {
	Reportsections []struct {
		ID           string `json:"id"`
		Reporttypeid string `json:"reportTypeId"`
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
	Mindt string `json:"minDt"`
	Maxdt string `json:"maxDt"`
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
	Totalstores   int `json:"totalStores"`
	Tradenetworks []struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		Storescount      int    `json:"storesCount"`
		Federaldistricts []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Storescount int    `json:"storesCount"`
			Regions     []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Storescount int    `json:"storesCount"`
				Cities      []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					Storescount int    `json:"storesCount"`
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

// ResultTreeProducts
type ResultTreeProducts struct {
	Nodes []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Level    string `json:"level"`
		Children []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Level    string `json:"level"`
			Children []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Level    string `json:"level"`
				Children []struct {
					ID       string        `json:"id"`
					Name     string        `json:"name"`
					Level    string        `json:"level"`
					Children []interface{} `json:"children"`
				} `json:"children"`
			} `json:"children"`
		} `json:"children"`
	} `json:"nodes"`
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
