package logistics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ReportService_Create(t *testing.T) {
	sdate, err := time.Parse("2006-01-02", "2021-01-15")
	assert.NoError(t, err)
	fdate, err := time.Parse("2006-01-02", "2021-01-15")
	assert.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, URL_REPORT_CREATE, r.URL.Path)

		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		var req RequestCreateReport
		err = json.Unmarshal(b, &req)
		assert.NoError(t, err)
		assert.Equal(t, req.FinishDate, fdate.Format(time.RFC3339))
		assert.Equal(t, req.StartDate, sdate.Format(time.RFC3339))
		assert.Equal(t, req.IsArchive, false)
		assert.Equal(t, req.SalesChannel, TSALL)
		assert.Equal(t, req.TypeReport, SALES)

		_, err = fmt.Fprintln(w, `{
			"code": "ok",
				"result": {
				"reportId": "8ff8998c-1eeb-412b-0000-000000000000"
			}
		}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
	})
	assert.NoError(t, err)

	req := RequestCreateReport{
		StartDate:    sdate.Format(time.RFC3339),
		FinishDate:   fdate.Format(time.RFC3339),
		SalesChannel: TSALL,
		TypeReport:   SALES,
		IsArchive:    false,
	}
	reportId, err := client.Reports.Create(req)
	assert.NoError(t, err)
	assert.Equal(t, "8ff8998c-1eeb-412b-0000-000000000000", reportId)
}

func Test_ReportService_Status_DONE(t *testing.T) {
	requestId := "8ff8998c-1eeb-412b-0000-000000000000"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf(URL_REPORT_STATUS, requestId), r.URL.Path)

		_, err := fmt.Fprintln(w, `{
			"code": "ok",
				"result": {
				"reportStatus": "DONE",
				"partIds": ["a369e2f7-0512-4229-0000-000000000000", "9199a162-7df6-47db-0000-000000000000"]
			}
		}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
	})
	assert.NoError(t, err)

	res, err := client.Reports.Status(requestId)
	assert.NoError(t, err)
	assert.Equal(t, DONE, res.Result.ReportStatus)
	want := []string{"a369e2f7-0512-4229-0000-000000000000", "9199a162-7df6-47db-0000-000000000000"}
	assert.Equal(t, want, res.Result.PartIds)
}

func Test_ReportService_Status_BUILD(t *testing.T) {
	requestId := "8ff8998c-1eeb-412b-0000-000000000000"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf(URL_REPORT_STATUS, requestId), r.URL.Path)

		_, err := fmt.Fprintln(w, `{
			"code": "ok",
			"result": {
				"requestId": "93cc7ae0-62f1-434e-0000-000000000000",
				"reportStatus": "BUILD",
				"description": "71%"
			}
		}`)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
	})
	assert.NoError(t, err)

	res, err := client.Reports.Status(requestId)
	assert.NoError(t, err)
	assert.Equal(t, BUILD, res.Result.ReportStatus)
}
