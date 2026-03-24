// Command logistics-reload is a CLI tool for the X5 Logistics reporting API.
// Unlike cmd/logistics, this command performs manual authentication — it calls
// Auth explicitly and then passes the obtained token to SetToken instead of
// relying on automatic (middleware-based) auth.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/germangorelkin/go-x5/internal/xconfig"
	"github.com/germangorelkin/go-x5/internal/xlog"
	"github.com/germangorelkin/go-x5/logistics"
	"go.uber.org/zap"
)

// main orchestrates the full report lifecycle:
//  1. Bootstrap logger and load configuration from environment variables.
//  2. Create a logistics API client.
//  3. Manually authenticate (Auth + SetToken) — no auto-auth middleware.
//  4. Submit a report creation request.
//  5. Poll the report status in a loop until it is DONE or ERROR (or attempts exhausted).
//  6. Download every report part to the configured output directory.
func main() {
	logger, verbose, err := xlog.Bootstrap("logistics-reload")
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
		zap.String("sales_channel", string(cfg.salesChannel)),
		zap.String("report_type", string(cfg.typeReport)),
		zap.String("out_dir", cfg.outDir),
	)
	logger.Info("command started")

	// Create the logistics API client without automatic auth.
	cli, err := logistics.NewClient(logistics.ClintConf{
		Instance: cfg.instance,
		Logger:   logger,
	})
	if err != nil {
		logger.Fatal("failed to build logistics client", zap.Error(err))
	}
	logger.Info("client created")

	// Manual auth: call Auth explicitly and inject the token into the client.
	token, err := cli.Auth.Auth(cfg.login, cfg.password)
	if err != nil {
		logger.Fatal("failed to authorize client", zap.Error(err))
	}
	cli.SetToken(token)
	logger.Info("manual authorization completed")

	// Build and submit the report creation request.
	reqCR := logistics.RequestCreateReport{
		StartDate:    cfg.startDate,
		FinishDate:   cfg.finishDAte,
		SalesChannel: cfg.salesChannel,
		TypeReport:   cfg.typeReport,
		IsArchive:    cfg.isArchive,
	}
	logger.Info("submitting report request",
		zap.String("start_date", reqCR.StartDate),
		zap.String("finish_date", reqCR.FinishDate),
		zap.Bool("archive", reqCR.IsArchive),
	)

	reportID, err := cli.Reports.Create(reqCR)
	if err != nil {
		logger.Fatal("failed to create report", zap.Error(err))
	}

	reportLog := logger.With(zap.String("report_id", reportID))
	reportLog.Info("report created")

	// Status polling loop: wait for the report to reach DONE or ERROR,
	// sleeping between each attempt for the configured delay.
	delay := time.Duration(cfg.waiteReportStatusDelaySec) * time.Second
	var resStatus logistics.ResponseStatusReport

	maxAttempts := cfg.waiteReportStatusAttempt + 1
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		reportLog.Info("waiting for report status",
			zap.Duration("delay", delay),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
		)
		time.Sleep(delay)

		resStatus, err = cli.Reports.Status(reportID)
		if err != nil {
			reportLog.Fatal("failed to fetch report status", zap.Error(err))
		}
		if resStatus.Result.ReportStatus == logistics.DONE ||
			resStatus.Result.ReportStatus == logistics.ERROR {
			break
		}
		reportLog.Info("report still processing",
			zap.String("status", string(resStatus.Result.ReportStatus)),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
		)
	}

	// Abort if the report did not finish successfully after all attempts.
	if resStatus.Result.ReportStatus != logistics.DONE {
		reportLog.Fatal("report generation did not finish successfully",
			zap.String("status", string(resStatus.Result.ReportStatus)),
			zap.String("description", resStatus.Result.Description),
		)
	}
	reportLog.Info("report ready", zap.Int("parts", len(resStatus.Result.PartIds)))

	// Download each report part to a local file in the output directory.
	for _, partID := range resStatus.Result.PartIds {
		path := filepath.Join(cfg.outDir, partID)
		partLog := reportLog.With(
			zap.String("part_id", partID),
			zap.String("path", path),
		)

		f, err := os.Create(path)
		if err != nil {
			partLog.Fatal("failed to create output file", zap.Error(err))
		}

		if err := cli.Reports.Download(partID, f); err != nil {
			partLog.Error("failed to download report part", zap.Error(err))
		} else {
			partLog.Info("report part downloaded")
		}

		if err := f.Close(); err != nil {
			partLog.Error("failed to close output file", zap.Error(err))
		}
	}

	logger.Info("command completed")
}

// config loads mainConfig from environment variables and applies sensible
// defaults where values are missing:
//   - OUT_DIR defaults to "reports"; the directory is created if absent.
//   - WAITE_REPORT_STATUS_DELAY_SEC defaults to 10 (seconds between polls).
//   - WAITE_REPORT_STATUS_ATTEMPT defaults to 10 polling attempts.
//   - START_DATE defaults to the current UTC date (truncated to midnight).
//   - FINISH_DATE defaults to the current UTC date minus 4 days.
//   - ARCHIVE defaults to false.
func config(verbose bool) (mainConfig, error) {
	cfg := mainConfig{
		instance:     os.Getenv("INSTANCE"),
		login:        os.Getenv("LOGIN"),
		password:     os.Getenv("PASSWORD"),
		verbose:      verbose,
		salesChannel: logistics.SalesChannel(os.Getenv("SALES_CHANNEL")),
		typeReport:   logistics.TypeReport(os.Getenv("TYPE_REPORT")),
		startDate:    os.Getenv("START_DATE"),
		finishDAte:   os.Getenv("FINISH_DATE"),
		outDir:       os.Getenv("OUT_DIR"),
	}

	var err error
	cfg.isArchive, err = xconfig.Bool("ARCHIVE", false)
	if err != nil {
		return cfg, err
	}
	cfg.waiteReportStatusDelaySec, err = xconfig.Int("WAITE_REPORT_STATUS_DELAY_SEC", 10)
	if err != nil {
		return cfg, err
	}
	cfg.waiteReportStatusAttempt, err = xconfig.Int("WAITE_REPORT_STATUS_ATTEMPT", 10)
	if err != nil {
		return cfg, err
	}

	if cfg.outDir == "" {
		cfg.outDir = "reports"
	}
	if err := os.MkdirAll(cfg.outDir, os.ModePerm); err != nil {
		return cfg, fmt.Errorf("failed to create out dir %s: %w", cfg.outDir, err)
	}

	if cfg.startDate == "" || cfg.finishDAte == "" {
		cfg.finishDAte = time.Now().UTC().Add(-4 * 24 * time.Hour).Truncate(24 * time.Hour).Format(time.RFC3339)
		cfg.startDate = time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	}

	return cfg, nil
}

// mainConfig holds all runtime configuration for the logistics-reload command.
// Values are populated from environment variables in the config function.
type mainConfig struct {
	instance                  string
	login                     string
	password                  string
	verbose                   bool
	salesChannel              logistics.SalesChannel
	typeReport                logistics.TypeReport
	startDate                 string // Defaults to the current UTC date if left empty.
	finishDAte                string // Defaults to current UTC date minus 4 days if left empty.
	outDir                    string // Defaults to "reports" if left empty.
	waiteReportStatusDelaySec int    // Defaults to 10 seconds if left empty.
	waiteReportStatusAttempt  int    // Defaults to 10 attempts if left empty.
	isArchive                 bool
}
