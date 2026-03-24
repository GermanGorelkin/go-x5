package insights

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportService_CreateTrends_OK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/public/reports/trends", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, `{
			"code": "ok",
			"result": {
				"id": "report-123",
				"reportTypeId": "type-1",
				"reportTypeName": "Trends Analysis",
				"reportName": "test-report",
				"reportStatusId": "status-1",
				"createdAt": "2024-01-01T00:00:00Z",
				"createdBy": "user-1",
				"accountId": "account-1",
				"parametersId": "params-1"
			}
		}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	result, err := client.Reports.CreateTrends(RequestTrendsAnalysis{
		Name:   "test-report",
		Type:   REPORT_TYPE_ID,
		Export: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "report-123", result.ID)
	assert.Equal(t, "type-1", result.ReportTypeID)
	assert.Equal(t, "Trends Analysis", result.ReportTypeName)
	assert.Equal(t, "test-report", result.ReportName)
	assert.Equal(t, "account-1", result.AccountID)
	assert.Equal(t, "params-1", result.ParametersID)
}

func TestReportService_CreateTrends_ErrorCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/public/reports/trends", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, `{"code":"error","result":{}}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	_, err = client.Reports.CreateTrends(RequestTrendsAnalysis{
		Name:   "test-report",
		Type:   REPORT_TYPE_ID,
		Export: true,
	})
	assert.Error(t, err)
}

func TestReportService_CreateTrends_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	_, err = client.Reports.CreateTrends(RequestTrendsAnalysis{
		Name:   "test-report",
		Type:   REPORT_TYPE_ID,
		Export: true,
	})
	assert.Error(t, err)
}

func TestReportService_GetReportStatus_OK(t *testing.T) {
	reportID := "report-abc-123"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/api/v2/public/reports/%s", reportID), r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, `{
			"code": "ok",
			"result": {
				"id": "report-abc-123",
				"type": "TRENDS_ANALYSIS",
				"name": "test-report",
				"status": "EXPORT_FILE_GENERATED",
				"deleted": false,
				"createdAt": "2024-01-01T00:00:00Z",
				"createdBy": "user-1",
				"accountId": "account-1",
				"parametersId": "params-1",
				"exportFileId": "export-file-456"
			}
		}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	result, err := client.Reports.GetReportStatus(reportID)
	require.NoError(t, err)
	assert.Equal(t, "report-abc-123", result.ID)
	assert.Equal(t, "EXPORT_FILE_GENERATED", result.Status)
	assert.Equal(t, "export-file-456", result.ExportFileID)
	assert.Equal(t, false, result.Deleted)
	assert.Equal(t, "account-1", result.AccountID)
}

func TestReportService_GetReportStatus_ErrorCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, `{"code":"not_found","result":{}}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	_, err = client.Reports.GetReportStatus("some-report-id")
	assert.Error(t, err)
}

func TestReportService_Download_OK(t *testing.T) {
	exportFileID := "export-file-789"
	data := "col1|col2|col3\nval1|val2|val3\nval4|val5|val6"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/api/v1/public/export/%s", exportFileID), r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprint(w, data)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	err = client.Reports.Download(exportFileID, buf)
	require.NoError(t, err)
	assert.Equal(t, data, buf.String())
}

func TestReportService_Download_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		KC_URL:   ts.URL,
		KC_RELM:  "test-realm",
		ClientID: "test-client",
		Login:    "user",
		Password: "pass",
		API_URL:  ts.URL,
	})
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	err = client.Reports.Download("some-export-id", buf)
	assert.Error(t, err)
}

func TestUniqueReportName_Month_Exclude(t *testing.T) {
	beginDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

	opts := TrendsAnalysisOptions{
		ReportType:   "TRENDS_ANALYSIS_DATA",
		PeriodMode:   PeriodMode_Month,
		DeliveryMode: DeliveryMode_EXCLUDE,
		BeginDate:    beginDate,
		EndDate:      endDate,
	}

	name := uniqueReportName(opts)

	expectedPrefix := "TRENDS_ANALYSIS_DATA-MONTH-EXCLUDE-20240115-20240630-"
	assert.True(t, strings.HasPrefix(name, expectedPrefix),
		"expected name to start with %q, got %q", expectedPrefix, name)

	// Verify that the name contains a nanosecond timestamp suffix
	parts := strings.Split(name, "-")
	assert.True(t, len(parts) >= 6, "expected at least 6 dash-separated parts, got %d", len(parts))
}

func TestUniqueReportName_Week_IncludeAll(t *testing.T) {
	beginDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)

	opts := TrendsAnalysisOptions{
		ReportType:   "TRENDS_ANALYSIS_WD",
		PeriodMode:   PeriodMode_Week,
		DeliveryMode: DeliveryMode_INCLUDE_ALL,
		BeginDate:    beginDate,
		EndDate:      endDate,
	}

	name := uniqueReportName(opts)

	assert.True(t, strings.Contains(name, "TRENDS_ANALYSIS_WD"),
		"expected name to contain report type, got %q", name)
	assert.True(t, strings.Contains(name, "WEEK"),
		"expected name to contain WEEK, got %q", name)
	assert.True(t, strings.Contains(name, "INCLUDE_ALL"),
		"expected name to contain INCLUDE_ALL, got %q", name)
	assert.True(t, strings.Contains(name, "20240301"),
		"expected name to contain begin date 20240301, got %q", name)
	assert.True(t, strings.Contains(name, "20240331"),
		"expected name to contain end date 20240331, got %q", name)

	expectedPrefix := "TRENDS_ANALYSIS_WD-WEEK-INCLUDE_ALL-20240301-20240331-"
	assert.True(t, strings.HasPrefix(name, expectedPrefix),
		"expected name to start with %q, got %q", expectedPrefix, name)
}
