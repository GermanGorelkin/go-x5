package insights

import "fmt"

const (
	REPORT_TYPE_ID = "8ddb5b9f-2193-453c-96ba-a0a3c14e517c" //
)

type ParametersService service

// Sections

// ResponseSections
type ResponseSections struct {
	Code   string `json:"code"`
	Result struct {
		Reportsections []struct {
			ID           string `json:"id"`
			Reporttypeid string `json:"reportTypeId"`
			Name         string `json:"name"`
		} `json:"reportSections"`
	} `json:"result"`
}

// GetSections returns all report sections for REPORT_TYPE_ID
func (srv *ParametersService) GetSections() (ResponseSections, error) {
	url := fmt.Sprintf(URL_BUILD_SECTIONS, srv.client.API_URL, REPORT_TYPE_ID)
	var res ResponseSections
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res, fmt.Errorf("failed to get sections: %v", err)
	}
	return res, nil
}

// AvailableDates

// ResponseAvailableDates
type ResponseAvailableDates struct {
	Code   string `json:"code"`
	Result struct {
		Mindt string `json:"minDt"`
		Maxdt string `json:"maxDt"`
	} `json:"result"`
}

// GetAvailableDates returns all available dates for REPORT_TYPE_ID
func (srv *ParametersService) GetAvailableDates() (ResponseAvailableDates, error) {
	url := fmt.Sprintf(URL_BUILD_AVAILABLE_DATE, srv.client.API_URL, REPORT_TYPE_ID)
	var res ResponseAvailableDates
	err := srv.client.httpClient.Get(url, &res)
	if err != nil || res.Code != "ok" {
		return res, fmt.Errorf("failed to get available dates: %v", err)
	}
	return res, nil
}
