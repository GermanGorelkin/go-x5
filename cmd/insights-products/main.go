// Command insights-products connects to the X5 Insights API, fetches the full
// product tree (report parameters), and exports all products as an XLSX file.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/germangorelkin/go-x5/insights"
	"github.com/germangorelkin/go-x5/internal/xconfig"
	"github.com/germangorelkin/go-x5/internal/xlog"
	"go.uber.org/zap"
)

func main() {
	// Initialize the structured logger with the service name.
	logger, verbose, err := xlog.Bootstrap("insights-products")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to bootstrap logger: %v\n", err)
		os.Exit(1)
	}
	defer xlog.Sync(logger)

	// Load and validate configuration from environment variables.
	cfg, err := config(verbose)
	if err != nil {
		logger.Fatal("invalid configuration", zap.Error(err))
	}

	logger = logger.With(
		zap.String("out_dir", cfg.OutDir),
	)
	logger.Info("command started")

	// Build the Insights API client with Keycloak and API credentials.
	cl, err := insights.NewClient(insights.ClintConf{
		KC_URL:   cfg.KC_URL,
		KC_RELM:  cfg.KC_RELM,
		API_URL:  cfg.API_URL,
		ClientID: cfg.ClientID,
		Login:    cfg.Login,
		Password: cfg.Password,
		Logger:   logger,
	})
	if err != nil {
		logger.Fatal("failed to build insights client", zap.Error(err))
	}
	logger.Info("client created")

	// Authenticate against the Keycloak identity provider.
	if err := cl.Authorization(); err != nil {
		logger.Fatal("authorization failed", zap.Error(err))
	}

	// Fetch the full product tree from the report parameters endpoint.
	parameters, err := cl.Parameters.FetchReportParameters()
	if err != nil {
		logger.Fatal("failed to fetch report parameters", zap.Error(err))
	}

	// Collect every product ID from the parameter tree for export.
	allProducts := parameters.GetAllProductIDs()
	logger.Info("resolved products for export", zap.Int("products", len(allProducts)))

	// Create the output XLSX file on disk.
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

	// Request the XLSX product export from the API and stream it to the file.
	rpd := insights.RequestProductsDownload{
		Nodes:         insights.ConvertToRequestProductsDownloadNode(allProducts),
		GlobalCatalog: false,
	}
	if err := cl.Parameters.ProductsDownload(rpd, fp); err != nil {
		logger.Fatal("failed to download products export", zap.Error(err))
	}

	logger.Info("products export downloaded", zap.String("path", fname))
}

// config reads environment variables and returns a validated mainConfig.
// It applies sensible defaults for optional fields (OutDir, delay, attempts)
// and ensures the output directory exists on disk.
func config(verbose bool) (mainConfig, error) {
	// Read all required and optional env vars into the config struct.
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

	// Parse optional numeric settings with their defaults.
	var err error
	cfg.WaiteReportStatusDelaySec, err = xconfig.Int("WAITE_REPORT_STATUS_DELAY_SEC", 60)
	if err != nil {
		return cfg, err
	}
	cfg.WaiteReportStatusAttempt, err = xconfig.Int("WAITE_REPORT_STATUS_ATTEMPT", 10)
	if err != nil {
		return cfg, err
	}

	// Fall back to "reports" when no output directory is specified.
	if cfg.OutDir == "" {
		cfg.OutDir = "reports"
	}
	if err := os.MkdirAll(cfg.OutDir, os.ModePerm); err != nil {
		return cfg, fmt.Errorf("failed to create out dir %s: %w", cfg.OutDir, err)
	}

	// Default the date window to today → 4 days ago (UTC) when not provided.
	if cfg.StartDate == "" || cfg.FinishDate == "" {
		cfg.FinishDate = time.Now().UTC().Add(-4 * 24 * time.Hour).Truncate(24 * time.Hour).Format(time.RFC3339)
		cfg.StartDate = time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	}

	return cfg, nil
}

// mainConfig holds all runtime settings for the insights-products command.
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
	OutDir                    string // Defaults to "reports" if left empty.
	WaiteReportStatusDelaySec int    // Defaults to 60 seconds if left empty.
	WaiteReportStatusAttempt  int    // Defaults to 10 attempts if left empty.
}
