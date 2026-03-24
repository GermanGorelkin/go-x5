// Command logistics creates a report on the X5 retail logistics platform,
// polls for its completion status, and downloads each report part to a
// local output directory. All configuration is supplied via environment
// variables (INSTANCE, LOGIN, PASSWORD, SALES_CHANNEL, TYPE_REPORT, etc.).
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

func main() {
	// Step 1: Bootstrap the structured logger; verbosity is derived from
	// environment/flag settings inside xlog.Bootstrap.
	logger, verbose, err := xlog.Bootstrap("logistics")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to bootstrap logger: %v\n", err)
		os.Exit(1)
	}
	defer xlog.Sync(logger)

	// Step 2: Load and validate the runtime configuration from environment
	// variables, applying sensible defaults where values are missing.
	cfg, err := config(verbose)
	if err != nil {
		logger.Fatal("invalid configuration", zap.Error(err))
	}

	// Enrich all subsequent log lines with the key config values.
	logger = logger.With(
		zap.String("sales_channel", string(cfg.salesChannel)),
		zap.String("report_type", string(cfg.typeReport)),
		zap.String("out_dir", cfg.outDir),
	)
	logger.Info("command started")

	// Step 3: Create an authenticated logistics API client.
	cli, err := logistics.NewClient(logistics.ClintConf{
		Instance: cfg.instance,
		Login:    cfg.login,
		Password: cfg.password,
		AutoAuth: cfg.autoAuth,
		Logger:   logger,
	})
	if err != nil {
		logger.Fatal("failed to build logistics client", zap.Error(err))
	}
	logger.Info("client created", zap.Bool("auto_auth", cfg.autoAuth))

	// Step 4: Submit a report creation request to the platform.
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

	// Step 5: Poll the report status in a loop until it reaches DONE or
	// ERROR, sleeping between each attempt.
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

	// Step 6: Verify the report finished successfully; abort otherwise.
	if resStatus.Result.ReportStatus != logistics.DONE {
		reportLog.Fatal("report generation did not finish successfully",
			zap.String("status", string(resStatus.Result.ReportStatus)),
			zap.String("description", resStatus.Result.Description),
		)
	}
	reportLog.Info("report ready", zap.Int("parts", len(resStatus.Result.PartIds)))

	// Step 7: Download each report part to the configured output directory.
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

// config reads environment variables and returns a populated mainConfig.
// It applies default values for optional fields and ensures the output
// directory exists on disk.
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
	cfg.autoAuth, err = xconfig.Bool("AUTO_AUTH", false)
	if err != nil {
		return cfg, err
	}
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

	// Default the output directory to "reports" when not specified.
	if cfg.outDir == "" {
		cfg.outDir = "reports"
	}
	if err := os.MkdirAll(cfg.outDir, os.ModePerm); err != nil {
		return cfg, fmt.Errorf("failed to create out dir %s: %w", cfg.outDir, err)
	}

	// When dates are omitted, default to a window from 4 days ago (finish)
	// to today (start). Both dates are truncated to midnight UTC.
	if cfg.startDate == "" || cfg.finishDAte == "" {
		cfg.finishDAte = time.Now().UTC().Add(-4 * 24 * time.Hour).Truncate(24 * time.Hour).Format(time.DateOnly)
		cfg.startDate = time.Now().UTC().Truncate(24 * time.Hour).Format(time.DateOnly)
	}

	return cfg, nil
}

// mainConfig holds all runtime settings for the logistics command.
type mainConfig struct {
	instance                  string                 // X5 platform instance URL.
	login                     string                 // Authentication login.
	password                  string                 // Authentication password.
	verbose                   bool                   // Enable verbose (debug) logging.
	autoAuth                  bool                   // Automatically re-authenticate on token expiry.
	salesChannel              logistics.SalesChannel // Sales channel filter for the report.
	typeReport                logistics.TypeReport   // Type of logistics report to generate.
	startDate                 string                 // Report window start date. If left empty, defaults to today (UTC).
	finishDAte                string                 // Report window finish date. If left empty, defaults to 4 days ago (UTC).
	outDir                    string                 // Directory for downloaded report files. Defaults to "reports".
	waiteReportStatusDelaySec int                    // Seconds to wait between status polling attempts. Defaults to 10.
	waiteReportStatusAttempt  int                    // Maximum number of status polling attempts. Defaults to 10.
	isArchive                 bool                   // Request the report in archive (compressed) format.
}
