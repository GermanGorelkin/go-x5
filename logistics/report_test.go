package logistics

import (
	"bytes"
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

func Test_ReportService_Download(t *testing.T) {
	partId := "d8eeb73b-80bc-4210-bf04-2ba3eb8f49887535676002360153464.csv"
	data := "DAY|PLANT|PLU|TURNOVER\n2021-06-09|1004|2070689|1.0000\n2021-06-09|1004|3357698|1.0000"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf(URL_REPORT_DOWNLOAD, partId), r.URL.Path)

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", "attachment;filename=d8eeb73b-80bc-4210-bf04-2ba3eb8f49887535676002360153464.csv")
		_, err := fmt.Fprintln(w, data)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client, err := NewClient(ClintConf{
		Instance: ts.URL,
	})
	assert.NoError(t, err)

	buf := bytes.NewBuffer([]byte{})
	err = client.Reports.Download(partId, buf)
	assert.NoError(t, err)
	assert.Equal(t, data+"\n", buf.String())
}
