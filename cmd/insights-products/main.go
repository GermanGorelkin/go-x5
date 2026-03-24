package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/germangorelkin/go-x5/insights"
	"github.com/germangorelkin/go-x5/internal/xlog"
	"go.uber.org/zap"
)

func main() {
	logger, verbose, err := xlog.Bootstrap("insights-products")
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

	allProducts := parameters.GetAllProductIDs()
	logger.Info("resolved products for export", zap.Int("products", len(allProducts)))

	fname := filepath.Join(cfg.OutDir, "products.xlsx")
	fp, err := os.Create(fname)
	if err != nil {
		logger.Fatal("failed to create output file", zap.String("path", fname), zap.Error(err))
	}
	defer func() {
		if err := fp.Close(); err != nil {
			logger.Error("failed to close output file", zap.String("path", fname), zap.Error(err))
		}
	}()

	rpd := insights.RequestProductsDownload{
		Nodes:         insights.ConvertToRequestProductsDownloadNode(allProducts),
		GlobalCatalog: false,
	}
	if err := cl.Parameters.ProductsDownload(rpd, fp); err != nil {
		logger.Fatal("failed to download products export", zap.Error(err))
	}

	logger.Info("products export downloaded", zap.String("path", fname))
}

func config(verbose bool) (mainConfig, error) {
	cfg := mainConfig{
		KC_URL:     os.Getenv("KC_URL"),
		KC_RELM:    os.Getenv("KC_RELM"),
		ClientID:   os.Getenv("CLIENT_ID"),
		Login:      os.Getenv("LOGIN"),
		Password:   os.Getenv("PASSWORD"),
		API_URL:    os.Getenv("API_URL"),
		Verbose:    verbose,
		StartDate:  os.Getenv("START_DATE"),
		FinishDate: os.Getenv("FINISH_DATE"),
		OutDir:     os.Getenv("OUT_DIR"),
	}

	var err error
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

	if cfg.StartDate == "" || cfg.FinishDate == "" {
		cfg.FinishDate = time.Now().UTC().Add(-4 * 24 * time.Hour).Truncate(24 * time.Hour).Format(time.RFC3339)
		cfg.StartDate = time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
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
	OutDir                    string // Если не заполнять поле то по умолчанию указывается report.
	WaiteReportStatusDelaySec int    // Если не заполнять поле то по умолчанию указывается 60 sec.
	WaiteReportStatusAttempt  int    // Если не заполнять поле то по умолчанию указывается 10.
}
