package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/germangorelkin/go-x5/internal/xlog"
	"github.com/germangorelkin/go-x5/logistics"
	"go.uber.org/zap"
)

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
		zap.String("instance", cfg.instance),
		zap.String("sales_channel", string(cfg.salesChannel)),
		zap.String("report_type", string(cfg.typeReport)),
		zap.String("out_dir", cfg.outDir),
	)
	logger.Info("command started")

	cli, err := logistics.NewClient(logistics.ClintConf{
		Instance: cfg.instance,
		Verbose:  cfg.verbose,
		Logger:   logger,
	})
	if err != nil {
		logger.Fatal("failed to build logistics client", zap.Error(err))
	}
	logger.Info("client created")

	token, err := cli.Auth.Auth(cfg.login, cfg.password)
	if err != nil {
		logger.Fatal("failed to authorize client", zap.Error(err))
	}
	cli.SetToken(token)
	logger.Info("manual authorization completed")

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

	if resStatus.Result.ReportStatus != logistics.DONE {
		reportLog.Fatal("report generation did not finish successfully",
			zap.String("status", string(resStatus.Result.ReportStatus)),
			zap.String("description", resStatus.Result.Description),
		)
	}
	reportLog.Info("report ready", zap.Int("parts", len(resStatus.Result.PartIds)))

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
	cfg.isArchive, err = parseBoolEnv("ARCHIVE", false)
	if err != nil {
		return cfg, err
	}
	cfg.waiteReportStatusDelaySec, err = parseIntEnv("WAITE_REPORT_STATUS_DELAY_SEC", 10)
	if err != nil {
		return cfg, err
	}
	cfg.waiteReportStatusAttempt, err = parseIntEnv("WAITE_REPORT_STATUS_ATTEMPT", 10)
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

func parseBoolEnv(key string, defaultValue bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s=%q: %w", key, value, err)
	}

	return parsed, nil
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

type mainConfig struct {
	instance                  string
	login                     string
	password                  string
	verbose                   bool
	salesChannel              logistics.SalesChannel
	typeReport                logistics.TypeReport
	startDate                 string // Если не заполнять поле то по умолчанию указывается текущая дата.
	finishDAte                string // Если не заполнять поле то по умолчанию указывается текущая дата -4 день.
	outDir                    string // Если не заполнять поле то по умолчанию указывается report.
	waiteReportStatusDelaySec int    // Если не заполнять поле то по умолчанию указывается 10 sec.
	waiteReportStatusAttempt  int    // Если не заполнять поле то по умолчанию указывается 10.
	isArchive                 bool
}
