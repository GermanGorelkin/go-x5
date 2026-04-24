package insights

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportParameters_SectionIDs(t *testing.T) {
	rp := ReportParameters{
		ResultSections: ResultSections{
			Reportsections: []struct {
				ID           string `json:"id"`
				ReportTypeID string `json:"reportTypeId"`
				Name         string `json:"name"`
			}{
				{ID: "sec-1", ReportTypeID: REPORT_TYPE_ID, Name: "Section 1"},
				{ID: "sec-2", ReportTypeID: "other-type-id", Name: "Section 2"},
				{ID: "sec-3", ReportTypeID: REPORT_TYPE_ID, Name: "Section 3"},
			},
		},
	}

	ids := rp.SectionIDs()

	assert.Len(t, ids, 2)
	assert.Contains(t, ids, "sec-1")
	assert.Contains(t, ids, "sec-3")
}

func TestReportParameters_SectionIDs_Empty(t *testing.T) {
	rp := ReportParameters{}

	ids := rp.SectionIDs()

	assert.Empty(t, ids)
}

func TestReportParameters_AvailableDates_OK(t *testing.T) {
	rp := ReportParameters{
		ResultAvailableDates: ResultAvailableDates{
			MinDT: "2024-01-15",
			MaxDT: "2024-06-30",
		},
	}

	minDT, maxDT, err := rp.AvailableDates()

	assert.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), minDT)
	assert.Equal(t, time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC), maxDT)
}

func TestReportParameters_AvailableDates_InvalidMin(t *testing.T) {
	rp := ReportParameters{
		ResultAvailableDates: ResultAvailableDates{
			MinDT: "not-a-date",
			MaxDT: "2024-06-30",
		},
	}

	_, _, err := rp.AvailableDates()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minDT")
}

func TestReportParameters_AvailableDates_InvalidMax(t *testing.T) {
	rp := ReportParameters{
		ResultAvailableDates: ResultAvailableDates{
			MinDT: "2024-01-15",
			MaxDT: "not-a-date",
		},
	}

	_, _, err := rp.AvailableDates()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maxDT")
}

func TestReportParameters_TradeNetworkIDs(t *testing.T) {
	rp := ReportParameters{
		ResultTreeStores: ResultTreeStores{
			TradeNetworks: []struct {
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
			}{
				{ID: "net-1", Name: "Network 1"},
				{ID: "net-2", Name: "Network 2"},
			},
		},
	}

	ids := rp.TradeNetworkIDs()

	assert.Len(t, ids, 2)
	assert.Equal(t, []string{"net-1", "net-2"}, ids)
}

func TestReportParameters_TradeNetworkIDs_Empty(t *testing.T) {
	rp := ReportParameters{}

	ids := rp.TradeNetworkIDs()

	assert.Empty(t, ids)
}

func TestReportParameters_ProductIDs(t *testing.T) {
	// Build a 4-level deep tree: L1 -> L2 -> L3 -> L4 (two leaves)
	rp := ReportParameters{
		ResultTreeProducts: ResultTreeProducts{
			Nodes: []TreeProductNodes{
				{
					ID:    "l1-1",
					Name:  "Level 1",
					Level: "Ui1",
					Children: []TreeProductNodes{
						{
							ID:    "l2-1",
							Name:  "Level 2",
							Level: "Ui2",
							Children: []TreeProductNodes{
								{
									ID:    "l3-1",
									Name:  "Level 3",
									Level: "Ui3",
									Children: []TreeProductNodes{
										{ID: "l4-1", Name: "Product A", Level: "Ui4"},
										{ID: "l4-2", Name: "Product B", Level: "Ui4"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ids := rp.ProductIDs()

	assert.Len(t, ids, 2)
	assert.Equal(t, ProductSectionID{Code: "l4-1", Level: "Ui4"}, ids[0])
	assert.Equal(t, ProductSectionID{Code: "l4-2", Level: "Ui4"}, ids[1])
}

func TestReportParameters_GetAllProductIDs(t *testing.T) {
	rp := ReportParameters{
		ResultTreeProducts: ResultTreeProducts{
			Nodes: []TreeProductNodes{
				{
					ID:    "l1-1",
					Name:  "Level 1",
					Level: "Ui1",
					Children: []TreeProductNodes{
						{
							ID:    "l2-1",
							Name:  "Level 2",
							Level: "Ui2",
							Children: []TreeProductNodes{
								{
									ID:    "l3-1",
									Name:  "Level 3",
									Level: "Ui3",
									Children: []TreeProductNodes{
										{ID: "l4-1", Name: "Product A", Level: "Ui4"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ids := rp.GetAllProductIDs()

	// Should include all levels: l1-1, l2-1, l3-1, l4-1
	assert.Len(t, ids, 4)

	codes := make([]string, len(ids))
	for i, id := range ids {
		codes[i] = id.Code
	}
	assert.Contains(t, codes, "l1-1")
	assert.Contains(t, codes, "l2-1")
	assert.Contains(t, codes, "l3-1")
	assert.Contains(t, codes, "l4-1")
}

func TestReportParameters_DeliveryIDs(t *testing.T) {
	rp := ReportParameters{
		ResultDelivery: ResultDelivery{
			Types: []struct {
				DeliveryTypeID   string `json:"deliveryTypeId"`
				DeliveryTypeName string `json:"deliveryTypeName"`
				Icon             string `json:"icon"`
				DateStart        string `json:"dateStart"`
			}{
				{DeliveryTypeID: "del-1", DeliveryTypeName: "Express", Icon: "icon1", DateStart: "2024-01-01"},
				{DeliveryTypeID: "del-2", DeliveryTypeName: "Standard", Icon: "icon2", DateStart: "2024-02-01"},
			},
		},
	}

	ids := rp.DeliveryIDs()

	assert.Len(t, ids, 2)
	assert.Equal(t, []string{"del-1", "del-2"}, ids)
}

func TestReportParameters_MetricIDs(t *testing.T) {
	rp := ReportParameters{
		ResultMetrics: ResultMetrics{
			MetricGroups: []struct {
				Code    string   `json:"code"`
				Metrics []string `json:"metrics"`
			}{
				{Code: "metric-group-1", Metrics: []string{"m1", "m2"}},
				{Code: "metric-group-2", Metrics: []string{"m3"}},
			},
		},
	}

	ids := rp.MetricIDs()

	assert.Len(t, ids, 2)
	assert.Equal(t, []string{"metric-group-1", "metric-group-2"}, ids)
}

func TestReportParameters_GranularityID_Found(t *testing.T) {
	rp := ReportParameters{
		ResultGranularities: ResultGranularities{
			Granularities: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				{ID: "gran-1", Name: "Daily"},
				{ID: "gran-2", Name: "Weekly"},
			},
		},
	}

	id := rp.GranularityID("Weekly")

	assert.Equal(t, "gran-2", id)
}

func TestReportParameters_GranularityID_NotFound(t *testing.T) {
	rp := ReportParameters{
		ResultGranularities: ResultGranularities{
			Granularities: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				{ID: "gran-1", Name: "Daily"},
				{ID: "gran-2", Name: "Weekly"},
			},
		},
	}

	id := rp.GranularityID("Monthly")

	assert.Equal(t, "", id)
}

func TestReportParameters_FederalDistricts(t *testing.T) {
	// We must construct ResultTreeStores with its anonymous struct types inline.
	rp := ReportParameters{}

	// Populate TradeNetworks with 2 networks, each having federal districts.
	// Network 1: FederalDistrict 1 with Region A; FederalDistrict 2 with Region B
	// Network 2: FederalDistrict 1 with Region A (duplicate); FederalDistrict 2 with Region C
	rp.ResultTreeStores.TradeNetworks = append(rp.ResultTreeStores.TradeNetworks, struct {
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
	}{
		ID:   "net-1",
		Name: "Network 1",
		FederalDistricts: []struct {
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
		}{
			{
				ID:   1,
				Name: "Central",
				Regions: []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					StoresCount int    `json:"storesCount"`
					Cities      []struct {
						ID          string `json:"id"`
						Name        string `json:"name"`
						StoresCount int    `json:"storesCount"`
					} `json:"cities"`
				}{
					{ID: "reg-a", Name: "Region A"},
				},
			},
			{
				ID:   2,
				Name: "North-West",
				Regions: []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					StoresCount int    `json:"storesCount"`
					Cities      []struct {
						ID          string `json:"id"`
						Name        string `json:"name"`
						StoresCount int    `json:"storesCount"`
					} `json:"cities"`
				}{
					{ID: "reg-b", Name: "Region B"},
				},
			},
		},
	})

	rp.ResultTreeStores.TradeNetworks = append(rp.ResultTreeStores.TradeNetworks, struct {
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
	}{
		ID:   "net-2",
		Name: "Network 2",
		FederalDistricts: []struct {
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
		}{
			// Same district+region as Network 1 — should be deduplicated
			{
				ID:   1,
				Name: "Central",
				Regions: []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					StoresCount int    `json:"storesCount"`
					Cities      []struct {
						ID          string `json:"id"`
						Name        string `json:"name"`
						StoresCount int    `json:"storesCount"`
					} `json:"cities"`
				}{
					{ID: "reg-a", Name: "Region A"},
				},
			},
			// Different region under existing district — should NOT be deduplicated
			{
				ID:   2,
				Name: "North-West",
				Regions: []struct {
					ID          string `json:"id"`
					Name        string `json:"name"`
					StoresCount int    `json:"storesCount"`
					Cities      []struct {
						ID          string `json:"id"`
						Name        string `json:"name"`
						StoresCount int    `json:"storesCount"`
					} `json:"cities"`
				}{
					{ID: "reg-c", Name: "Region C"},
				},
			},
		},
	})

	result := rp.FederalDistricts()

	// Expected: 3 unique (districtID, regionID) pairs:
	// (1, reg-a), (2, reg-b), (2, reg-c)
	// The duplicate (1, reg-a) from Network 2 should be removed.
	assert.Len(t, result, 3)

	// Verify each entry has SelectedFully=false and its region has SelectedFully=true
	for _, fd := range result {
		assert.False(t, fd.SelectedFully)
		assert.Len(t, fd.Regions, 1)
		assert.True(t, fd.Regions[0].SelectedFully)
		assert.Equal(t, []string{}, fd.Regions[0].CitiesID)
	}

	// Check the unique entries exist
	type key struct {
		districtID int
		regionID   string
	}
	seen := make(map[key]bool)
	for _, fd := range result {
		seen[key{fd.DistrictID, fd.Regions[0].RegionID}] = true
	}
	assert.True(t, seen[key{1, "reg-a"}], "expected (1, reg-a)")
	assert.True(t, seen[key{2, "reg-b"}], "expected (2, reg-b)")
	assert.True(t, seen[key{2, "reg-c"}], "expected (2, reg-c)")
}

func TestConvertToRequestProductsDownloadNode(t *testing.T) {
	src := []ProductSectionID{
		{Code: "prod-1", Level: "Ui4"},
		{Code: "prod-2", Level: "Ui3"},
	}

	result := ConvertToRequestProductsDownloadNode(src)

	assert.Len(t, result, 2)
	assert.Equal(t, RequestProductsDownloadNode{ID: "prod-1", Level: "Ui4"}, result[0])
	assert.Equal(t, RequestProductsDownloadNode{ID: "prod-2", Level: "Ui3"}, result[1])
}

func TestConvertToRequestProductsDownloadNode_Empty(t *testing.T) {
	src := []ProductSectionID{}

	result := ConvertToRequestProductsDownloadNode(src)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestParametersService_ProductsDownload_UsesAuthorizationHeaders(t *testing.T) {
	const realm = "test-realm"

	var internalTokenCalls int
	var productsDownloadCalls int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/auth/realms/%s/protocol/openid-connect/token", realm):
			_, err := fmt.Fprint(w, `{"access_token":"access-1","expires_in":300,"refresh_expires_in":1800,"refresh_token":"refresh-1"}`)
			require.NoError(t, err)
		case "/api/v1/public/auth/token":
			internalTokenCalls++
			assert.Equal(t, "Bearer access-1", r.Header.Get("Authorization"))
			_, err := fmt.Fprint(w, `{"code":"ok","result":{"token":"jwt-1"}}`)
			require.NoError(t, err)
		case "/api/v1/public/tree/products/download":
			productsDownloadCalls++
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "Bearer access-1", r.Header.Get("Authorization"))
			assert.Equal(t, "jwt-1", r.Header.Get("x5-api-key"))

			var req RequestProductsDownload
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, []RequestProductsDownloadNode{{ID: "prod-1", Level: "Ui4"}}, req.Nodes)
			assert.False(t, req.GlobalCatalog)

			_, err := fmt.Fprint(w, "xlsx-content")
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  realm,
		ClientID: "client-id",
		Login:    "login",
		Password: "password",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	require.NoError(t, client.Authorization())
	require.NoError(t, client.Authorization())

	var out bytes.Buffer
	err = client.Parameters.ProductsDownload(RequestProductsDownload{
		Nodes:         []RequestProductsDownloadNode{{ID: "prod-1", Level: "Ui4"}},
		GlobalCatalog: false,
	}, &out)
	require.NoError(t, err)

	assert.Equal(t, "xlsx-content", out.String())
	assert.Equal(t, 1, internalTokenCalls)
	assert.Equal(t, 1, productsDownloadCalls)
}
