package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/germangorelkin/go-x5/insights"
)

func main() {
	cfg := config()

	cl, err := insights.NewClient(insights.ClintConf{
		KC_URL:   cfg.KC_URL,
		KC_RELM:  cfg.KC_RELM,
		API_URL:  cfg.API_URL,
		ClientID: cfg.ClientID,
		Login:    cfg.Login,
		Password: cfg.Password,
		Verbose:  false,
	})
	if err != nil {
		panic(err)
	}
	log.Println("build new client")

	if err := cl.Authorization(); err != nil {
		panic(err)
	}

	parameters, err := cl.Parameters.FetchReportParameters()
	if err != nil {
		log.Fatalf("Error FetchReportParameters:%v", err)
	}

	log.Println("fetch ReportParameters")

	// --------- build requests
	var beginDate, endDate time.Time

	if cfg.StartDate != "" && cfg.FinishDate != "" {
		beginDate, err = time.Parse("2006-01-02", cfg.StartDate)
		if err != nil {
			log.Printf("failed to parse %s:%v", cfg.StartDate, err)
		}
		endDate, err = time.Parse("2006-01-02", cfg.FinishDate)
		if err != nil {
			log.Printf("failed to parse %s:%v", cfg.FinishDate, err)
		}
	}
	if beginDate.IsZero() || endDate.IsZero() {
		beginDate, endDate = getPeriod()
	}

	beginWeekDate := getMonday(beginDate)
	endWeekDate := getSunday(endDate)

	requests := make([]insights.RequestTrendsAnalysis, 0, 6)

	/*
		до 1 марта только INCLUDE_ALL
	*/

	// TRENDS_ANALYSIS_DATA + Week + CHOOSE_ONLY_DELIVERY
	opts := insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Week,
		DeliveryMode:       insights.DeliveryMode_CHOOSE_ONLY_DELIVERY,
		BeginDate:          beginWeekDate,
		EndDate:            endWeekDate,
		GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
		ReportType:         "TRENDS_ANALYSIS_DATA",
	}
	req, err := cl.Reports.BuildRequestTrendsAnalysis(opts)
	if err != nil {
		panic(err)
	}
	requests = append(requests, req)

	// TRENDS_ANALYSIS_DATA + Week + EXCLUDE
	opts = insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Week,
		DeliveryMode:       insights.DeliveryMode_EXCLUDE,
		BeginDate:          beginWeekDate,
		EndDate:            endWeekDate,
		GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
		ReportType:         "TRENDS_ANALYSIS_DATA",
	}
	req, err = cl.Reports.BuildRequestTrendsAnalysis(opts)
	if err != nil {
		panic(err)
	}
	requests = append(requests, req)

	// TRENDS_ANALYSIS_DATA + Week + INCLUDE_ALL
	// opts := insights.TrendsAnalysisOptions{
	// 	Params:       parameters,
	// 	PeriodMode:   insights.PeriodMode_Week,
	// 	DeliveryMode: insights.DeliveryMode_INCLUDE_ALL,
	// 	BeginDate:    beginWeekDate,
	// 	EndDate:      endWeekDate,
	// 	ReportType:   "TRENDS_ANALYSIS_DATA",
	// }
	// req, err := cl.Reports.BuildRequestTrendsAnalysis(opts)
	// if err != nil {
	// 	panic(err)
	// }
	// requests = append(requests, req)

	// TRENDS_ANALYSIS_WD + Week + INCLUDE_ALL
	opts = insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Week,
		DeliveryMode:       insights.DeliveryMode_INCLUDE_ALL,
		BeginDate:          beginWeekDate,
		EndDate:            endWeekDate,
		GroupingAttributes: []string{"TRADE_NETWORK"},
		ReportType:         "TRENDS_ANALYSIS_WD",
	}
	req, err = cl.Reports.BuildRequestTrendsAnalysis(opts)
	if err != nil {
		panic(err)
	}
	requests = append(requests, req)

	// TRENDS_ANALYSIS_DATA + Month + CHOOSE_ONLY_DELIVERY
	opts = insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Month,
		DeliveryMode:       insights.DeliveryMode_CHOOSE_ONLY_DELIVERY,
		BeginDate:          beginDate,
		EndDate:            endDate,
		GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
		ReportType:         "TRENDS_ANALYSIS_DATA",
	}
	req, err = cl.Reports.BuildRequestTrendsAnalysis(opts)
	if err != nil {
		panic(err)
	}
	requests = append(requests, req)

	// TRENDS_ANALYSIS_DATA + Month + EXCLUDE
	opts = insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Month,
		DeliveryMode:       insights.DeliveryMode_EXCLUDE,
		BeginDate:          beginDate,
		EndDate:            endDate,
		GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
		ReportType:         "TRENDS_ANALYSIS_DATA",
	}
	req, err = cl.Reports.BuildRequestTrendsAnalysis(opts)
	if err != nil {
		panic(err)
	}
	requests = append(requests, req)

	// TRENDS_ANALYSIS_DATA + Month + INCLUDE_ALL
	// opts = insights.TrendsAnalysisOptions{
	// 	Params:       parameters,
	// 	PeriodMode:   insights.PeriodMode_Month,
	// 	DeliveryMode: insights.DeliveryMode_INCLUDE_ALL,
	// 	BeginDate:    beginDate,
	// 	EndDate:      endDate,
	// 	ReportType:   "TRENDS_ANALYSIS_DATA",
	// }
	// req, err = cl.Reports.BuildRequestTrendsAnalysis(opts)
	// if err != nil {
	// 	panic(err)
	// }
	// requests = append(requests, req)

	// TRENDS_ANALYSIS_WD + Month + INCLUDE_ALL
	opts = insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Month,
		DeliveryMode:       insights.DeliveryMode_INCLUDE_ALL,
		BeginDate:          beginDate,
		EndDate:            endDate,
		GroupingAttributes: []string{"TRADE_NETWORK"},
		ReportType:         "TRENDS_ANALYSIS_WD",
	}
	req, err = cl.Reports.BuildRequestTrendsAnalysis(opts)
	if err != nil {
		panic(err)
	}
	requests = append(requests, req)
	// ---------

	// ------

	if err := os.MkdirAll("reports", os.ModePerm); err != nil {
		log.Fatal(err)
	}

	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup

	for _, reqReport := range requests {
		wg.Add(1)
		go func(reqReport insights.RequestTrendsAnalysis) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			reportName := reqReport.Name

			jsonData, err := json.Marshal(reqReport)
			if err != nil {
				panic(err)
			}

			err = os.WriteFile(fmt.Sprintf("reports/%s.json", reportName), jsonData, 0644)
			if err != nil {
				panic(err)
			}
			log.Printf("save json:%v", string(jsonData))

			res, err := cl.Reports.CreateTrends(reqReport)
			if err != nil {
				panic(err)
			}
			log.Printf("res:%v", res)

			var status insights.ResultReportStatus
			var delay time.Duration
			for attempts := 0; attempts < 10; attempts++ {
				status, err = cl.Reports.GetReportStatus(res.ID)
				if err != nil {
					panic(err)
				}

				log.Printf("status:%v", status)

				if status.Status == "SUCCEEDED" || status.Status == "FAILED" {
					break
				}

				delay += 5 * time.Minute
				log.Printf("wait %s", delay)
				time.Sleep(delay)

				if err := cl.Authorization(); err != nil {
					panic(err)
				}
			}

			if status.Status == "FAILED" {
				log.Fatalf("failed to create report:%v", status)
			}

			f, err := os.Create(fmt.Sprintf("reports/%s.zip", reportName))
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			delay = 0
			for attempts := 0; attempts < 10; attempts++ {
				err := cl.Reports.Download(status.ExportFileID, f)
				if err == nil {
					log.Println("download report")
					break
				}

				log.Printf("failed to download:%v", err)
				delay += 5 * time.Minute
				log.Printf("wait %s", delay)
				time.Sleep(delay)
			}
		}(reqReport)
	}

	wg.Wait()

}

func config() mainConfig {
	var cfg mainConfig

	cfg.KC_URL = os.Getenv("KC_URL")
	cfg.KC_RELM = os.Getenv("KC_RELM")
	cfg.ClientID = os.Getenv("CLIENT_ID")
	cfg.Login = os.Getenv("LOGIN")
	cfg.Password = os.Getenv("PASSWORD")
	cfg.API_URL = os.Getenv("API_URL")
	cfg.Verbose, _ = strconv.ParseBool(os.Getenv("VERBOSE"))
	cfg.StartDate = os.Getenv("START_DATE")
	cfg.FinishDate = os.Getenv("FINISH_DATE")
	cfg.OutDir = os.Getenv("OUT_DIR")

	waiteReportStatusDelaySec := os.Getenv("WAITE_REPORT_STATUS_DELAY_SEC")
	waiteReportStatusAttempt := os.Getenv("WAITE_REPORT_STATUS_ATTEMPT")

	if waiteReportStatusDelaySec == "" {
		waiteReportStatusDelaySec = "60"
	}
	n, err := strconv.Atoi(waiteReportStatusDelaySec)
	if err != nil {
		log.Fatalf("failed to parse WAITE_REPORT_STATUS_DELAY_SEC=%s", waiteReportStatusDelaySec)
	}
	cfg.WaiteReportStatusDelaySec = n

	if waiteReportStatusAttempt == "" {
		waiteReportStatusAttempt = "10"
	}
	n, err = strconv.Atoi(waiteReportStatusAttempt)
	if err != nil {
		log.Fatalf("failed to parse WAITE_REPORT_STATUS_ATTEMPT=%s", waiteReportStatusAttempt)
	}
	cfg.WaiteReportStatusAttempt = n

	if cfg.OutDir == "" {
		cfg.OutDir = "reports"
	}
	if err := os.MkdirAll(cfg.OutDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create out dir %s:%v", cfg.OutDir, err)
	}

	return cfg
}

// getPeriod gets period form prev month to curr month
func getPeriod() (begin, end time.Time) {
	now := time.Now().UTC()
	firstOfMonth := now.AddDate(0, 0, -now.Day()+1) // day=1
	prevMonth := firstOfMonth.AddDate(0, -1, 0)     // -1 month
	return prevMonth, now
}

// getMonday gets Monday of the current week
func getMonday(dt time.Time) time.Time {
	mon := dt
	for mon.Weekday() != time.Monday {
		mon = mon.AddDate(0, 0, -1)
	}
	return mon
}

func getSunday(dt time.Time) time.Time {
	d := dt
	for d.Weekday() != time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

type mainConfig struct {
	KC_URL   string
	KC_RELM  string
	ClientID string
	Login    string
	Password string
	API_URL  string

	Verbose                   bool
	StartDate                 string
	FinishDate                string
	OutDir                    string //  Если не заполнять поле то по умолчанию указывается report
	WaiteReportStatusDelaySec int    //  Если не заполнять поле то по умолчанию указывается 60 sec
	WaiteReportStatusAttempt  int    //  Если не заполнять поле то по умолчанию указывается 10
}
