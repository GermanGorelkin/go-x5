package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/germangorelkin/go-x5/insights"
	"github.com/germangorelkin/go-x5/internal/xlog"
	"go.uber.org/zap"
)

func main() {
	logger, verbose, err := xlog.Bootstrap("insights")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to bootstrap logger: %v\n", err)
		os.Exit(1)
	}
	defer xlog.Sync(logger)

	cfg, err := config(verbose)
	if err != nil {
		logger.Fatal("invalid configuration", zap.Error(err))
	}

	logger = logger.With(
		zap.String("api_url", cfg.API_URL),
		zap.String("out_dir", cfg.OutDir),
		zap.Int("group_request", cfg.GroupRequest),
	)
	logger.Info("command started")

	cl, err := insights.NewClient(insights.ClintConf{
		KC_URL:   cfg.KC_URL,
		KC_RELM:  cfg.KC_RELM,
		API_URL:  cfg.API_URL,
		ClientID: cfg.ClientID,
		Login:    cfg.Login,
		Password: cfg.Password,
		Verbose:  cfg.Verbose,
		Logger:   logger,
	})
	if err != nil {
		logger.Fatal("failed to build insights client", zap.Error(err))
	}
	logger.Info("client created")

	if err := cl.Authorization(); err != nil {
		logger.Fatal("authorization failed", zap.Error(err))
	}

	parameters, err := cl.Parameters.FetchReportParameters()
	if err != nil {
		logger.Fatal("failed to fetch report parameters", zap.Error(err))
	}

	beginDate, endDate := resolvePeriod(
		logger.Named("dates"),
		cfg.StartDate,
		cfg.FinishDate,
		getPeriod,
		"month",
	)
	beginWeekDate, endWeekDate := resolvePeriod(
		logger.Named("dates"),
		cfg.StartWeekDate,
		cfg.FinishWeekDate,
		func() (time.Time, time.Time) {
			return getMonday(beginDate), getSunday(endDate)
		},
		"week",
	)

	requests, err := buildRequests(cl, parameters, cfg.GroupRequest, beginDate, endDate, beginWeekDate, endWeekDate)
	if err != nil {
		logger.Fatal("failed to build report requests", zap.Error(err))
	}
	logger.Info("report requests prepared", zap.Int("count", len(requests)))

	numReq := 3
	if cfg.GroupRequest == 2 {
		numReq = 4
	}

	sem := make(chan struct{}, numReq)
	errCh := make(chan error, len(requests))

	var wg sync.WaitGroup
	for _, reqReport := range requests {
		wg.Add(1)
		time.Sleep(5 * time.Second)

		go func(reqReport insights.RequestTrendsAnalysis) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			reportLog := logger.Named("report-job").With(
				zap.String("report_name", reqReport.Name),
				zap.String("request_type", reqReport.Type),
			)
			if err := runReport(reportLog, cl, cfg.OutDir, reqReport); err != nil {
				errCh <- err
			}
		}(reqReport)
	}

	wg.Wait()
	close(errCh)

	var firstErr error
	for err := range errCh {
		logger.Error("report job failed", zap.Error(err))
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		logger.Fatal("one or more report jobs failed", zap.Error(firstErr))
	}

	logger.Info("command completed")
}

func buildRequests(
	cl *insights.Client,
	parameters insights.ReportParameters,
	groupRequest int,
	beginDate, endDate, beginWeekDate, endWeekDate time.Time,
) ([]insights.RequestTrendsAnalysis, error) {
	requests := make([]insights.RequestTrendsAnalysis, 0, 6)

	addRequest := func(opts insights.TrendsAnalysisOptions) error {
		req, err := cl.Reports.BuildRequestTrendsAnalysis(opts)
		if err != nil {
			return err
		}
		requests = append(requests, req)
		return nil
	}

	if groupRequest == 1 {
		if err := addRequest(insights.TrendsAnalysisOptions{
			Params:             parameters,
			PeriodMode:         insights.PeriodMode_Week,
			DeliveryMode:       insights.DeliveryMode_CHOOSE_ONLY_DELIVERY,
			BeginDate:          beginWeekDate,
			EndDate:            endWeekDate,
			GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
			ReportType:         "TRENDS_ANALYSIS_DATA",
		}); err != nil {
			return nil, err
		}
		if err := addRequest(insights.TrendsAnalysisOptions{
			Params:             parameters,
			PeriodMode:         insights.PeriodMode_Week,
			DeliveryMode:       insights.DeliveryMode_EXCLUDE,
			BeginDate:          beginWeekDate,
			EndDate:            endWeekDate,
			GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
			ReportType:         "TRENDS_ANALYSIS_DATA",
		}); err != nil {
			return nil, err
		}
	} else if groupRequest == 2 {
		if err := addRequest(insights.TrendsAnalysisOptions{
			Params:             parameters,
			PeriodMode:         insights.PeriodMode_Week,
			DeliveryMode:       insights.DeliveryMode_INCLUDE_ALL,
			BeginDate:          beginWeekDate,
			EndDate:            endWeekDate,
			GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
			ReportType:         "TRENDS_ANALYSIS_DATA",
		}); err != nil {
			return nil, err
		}
	}

	if err := addRequest(insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Week,
		DeliveryMode:       insights.DeliveryMode_INCLUDE_ALL,
		BeginDate:          beginWeekDate,
		EndDate:            endWeekDate,
		GroupingAttributes: []string{"TRADE_NETWORK"},
		ReportType:         "TRENDS_ANALYSIS_WD",
	}); err != nil {
		return nil, err
	}

	if groupRequest == 1 {
		if err := addRequest(insights.TrendsAnalysisOptions{
			Params:             parameters,
			PeriodMode:         insights.PeriodMode_Month,
			DeliveryMode:       insights.DeliveryMode_CHOOSE_ONLY_DELIVERY,
			BeginDate:          beginDate,
			EndDate:            endDate,
			GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
			ReportType:         "TRENDS_ANALYSIS_DATA",
		}); err != nil {
			return nil, err
		}
		if err := addRequest(insights.TrendsAnalysisOptions{
			Params:             parameters,
			PeriodMode:         insights.PeriodMode_Month,
			DeliveryMode:       insights.DeliveryMode_EXCLUDE,
			BeginDate:          beginDate,
			EndDate:            endDate,
			GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
			ReportType:         "TRENDS_ANALYSIS_DATA",
		}); err != nil {
			return nil, err
		}
	} else {
		if err := addRequest(insights.TrendsAnalysisOptions{
			Params:             parameters,
			PeriodMode:         insights.PeriodMode_Month,
			DeliveryMode:       insights.DeliveryMode_INCLUDE_ALL,
			BeginDate:          beginDate,
			EndDate:            endDate,
			GroupingAttributes: []string{"TRADE_NETWORK", "FEDERAL_DISTRICT", "REGION", "CITY"},
			ReportType:         "TRENDS_ANALYSIS_DATA",
		}); err != nil {
			return nil, err
		}
	}

	if err := addRequest(insights.TrendsAnalysisOptions{
		Params:             parameters,
		PeriodMode:         insights.PeriodMode_Month,
		DeliveryMode:       insights.DeliveryMode_INCLUDE_ALL,
		BeginDate:          beginDate,
		EndDate:            endDate,
		GroupingAttributes: []string{"TRADE_NETWORK"},
		ReportType:         "TRENDS_ANALYSIS_WD",
	}); err != nil {
		return nil, err
	}

	return requests, nil
}

func runReport(logger *zap.Logger, cl *insights.Client, outDir string, reqReport insights.RequestTrendsAnalysis) error {
	logger.Info("starting report job")

	jsonData, err := json.Marshal(reqReport)
	if err != nil {
		return fmt.Errorf("failed to marshal request %s: %w", reqReport.Name, err)
	}

	jsonPath := filepath.Join(outDir, reqReport.Name+".json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0755); err != nil {
		return fmt.Errorf("failed to create request directory %s: %w", filepath.Dir(jsonPath), err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write request file %s: %w", jsonPath, err)
	}
	logger.Info("saved request payload",
		zap.String("path", jsonPath),
		zap.Int("bytes", len(jsonData)),
	)

	if err := cl.Authorization(); err != nil {
		return fmt.Errorf("failed to authorize before create for %s: %w", reqReport.Name, err)
	}
	res, err := cl.Reports.CreateTrends(reqReport)
	if err != nil {
		return fmt.Errorf("failed to create trends report %s: %w", reqReport.Name, err)
	}

	reportLog := logger.With(zap.String("report_id", res.ID))
	reportLog.Info("report job created")

	var status insights.ResultReportStatus
	for attempt := 1; attempt <= 36; attempt++ { // 36 * 5min = 3h
		if err := cl.Authorization(); err != nil {
			return fmt.Errorf("failed to authorize before status check for %s: %w", reqReport.Name, err)
		}

		status, err = cl.Reports.GetReportStatus(res.ID)
		if err != nil {
			return fmt.Errorf("failed to get report status for %s: %w", reqReport.Name, err)
		}

		reportLog.Info("report status received",
			zap.String("status", status.Status),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", 36),
		)

		if status.Status == "SUCCEEDED" || status.Status == "FAILED" {
			break
		}

		delay := 5 * time.Minute
		reportLog.Info("waiting before next status check", zap.Duration("delay", delay))
		time.Sleep(delay)
	}

	if status.Status == "FAILED" {
		return fmt.Errorf("report %s failed to generate: %+v", reqReport.Name, status)
	}
	if status.Status != "SUCCEEDED" {
		return fmt.Errorf("report %s did not reach terminal success state: %+v", reqReport.Name, status)
	}

	reportPath := filepath.Join(outDir, reqReport.Name+".zip")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0755); err != nil {
		return fmt.Errorf("failed to create report directory %s: %w", filepath.Dir(reportPath), err)
	}

	f, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create report file %s: %w", reportPath, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			reportLog.Error("failed to close report file", zap.String("path", reportPath), zap.Error(closeErr))
		}
	}()

	var downloadErr error
	for attempt := 1; attempt <= 5; attempt++ {
		if attempt > 1 {
			if err := f.Truncate(0); err != nil {
				return fmt.Errorf("failed to truncate report file %s: %w", reportPath, err)
			}
			if _, err := f.Seek(0, 0); err != nil {
				return fmt.Errorf("failed to rewind report file %s: %w", reportPath, err)
			}
		}

		if err := cl.Authorization(); err != nil {
			return fmt.Errorf("failed to authorize before download for %s: %w", reqReport.Name, err)
		}
		if err := cl.Reports.Download(status.ExportFileID, f); err == nil {
			reportLog.Info("report downloaded", zap.String("path", reportPath))
			return nil
		} else {
			downloadErr = err
		}

		delay := 5 * time.Minute
		reportLog.Warn("failed to download report, will retry",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", 5),
			zap.Duration("delay", delay),
			zap.Error(downloadErr),
		)
		time.Sleep(delay)
	}

	return fmt.Errorf("failed to download report %s after retries: %w", reqReport.Name, downloadErr)
}

func config(verbose bool) (mainConfig, error) {
	cfg := mainConfig{
		KC_URL:         os.Getenv("KC_URL"),
		KC_RELM:        os.Getenv("KC_RELM"),
		ClientID:       os.Getenv("CLIENT_ID"),
		Login:          os.Getenv("LOGIN"),
		Password:       os.Getenv("PASSWORD"),
		API_URL:        os.Getenv("API_URL"),
		Verbose:        verbose,
		StartDate:      os.Getenv("START_DATE"),
		FinishDate:     os.Getenv("FINISH_DATE"),
		StartWeekDate:  os.Getenv("START_WEEK_DATE"),
		FinishWeekDate: os.Getenv("FINISH_WEEK_DATE"),
		OutDir:         os.Getenv("OUT_DIR"),
	}

	var err error
	cfg.GroupRequest, err = parseIntEnv("GROUP_REQUEST", 1)
	if err != nil {
		return cfg, err
	}
	if cfg.GroupRequest != 1 && cfg.GroupRequest != 2 {
		return cfg, fmt.Errorf("GROUP_REQUEST must be 1 or 2, got %d", cfg.GroupRequest)
	}

	cfg.WaiteReportStatusDelaySec, err = parseIntEnv("WAITE_REPORT_STATUS_DELAY_SEC", 60)
	if err != nil {
		return cfg, err
	}
	cfg.WaiteReportStatusAttempt, err = parseIntEnv("WAITE_REPORT_STATUS_ATTEMPT", 10)
	if err != nil {
		return cfg, err
	}

	if cfg.OutDir == "" {
		cfg.OutDir = "reports"
	}
	if err := os.MkdirAll(cfg.OutDir, os.ModePerm); err != nil {
		return cfg, fmt.Errorf("failed to create out dir %s: %w", cfg.OutDir, err)
	}

	return cfg, nil
}

func parseIntEnv(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s=%q: %w", key, value, err)
	}

	return parsed, nil
}

func resolvePeriod(
	logger *zap.Logger,
	startValue, finishValue string,
	fallback func() (time.Time, time.Time),
	periodKind string,
) (time.Time, time.Time) {
	var (
		beginDate time.Time
		endDate   time.Time
		err       error
	)

	if startValue != "" && finishValue != "" {
		beginDate, err = time.Parse(time.DateOnly, startValue)
		if err != nil {
			logger.Warn("failed to parse period start, using fallback",
				zap.String("period_kind", periodKind),
				zap.String("value", startValue),
				zap.Error(err),
			)
		}

		endDate, err = time.Parse(time.DateOnly, finishValue)
		if err != nil {
			logger.Warn("failed to parse period end, using fallback",
				zap.String("period_kind", periodKind),
				zap.String("value", finishValue),
				zap.Error(err),
			)
		}
	}

	if beginDate.IsZero() || endDate.IsZero() {
		beginDate, endDate = fallback()
		logger.Info("using fallback period",
			zap.String("period_kind", periodKind),
			zap.String("begin_date", beginDate.Format(time.DateOnly)),
			zap.String("end_date", endDate.Format(time.DateOnly)),
		)
	}

	return beginDate, endDate
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
	StartWeekDate             string
	FinishWeekDate            string
	OutDir                    string // Если не заполнять поле то по умолчанию указывается report.
	WaiteReportStatusDelaySec int    // Если не заполнять поле то по умолчанию указывается 60 sec.
	WaiteReportStatusAttempt  int    // Если не заполнять поле то по умолчанию указывается 10.

	GroupRequest int // 1 (default) - CHOOSE_ONLY_DELIVERY + EXCLUDE + INCLUDE_ALL(WD); 2 - INCLUDE_ALL + INCLUDE_ALL(WD)
}
