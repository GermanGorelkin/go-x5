package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
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

	allProducts := parameters.GetAllProductIDs()

	fname := filepath.Join(cfg.OutDir, "products.xlsx")
	fp, err := os.Create(fname)
	if err != nil {
		log.Fatalf("Error create file %q:%v", fname, err)
	}
	defer fp.Close()

	rpd := insights.RequestProductsDownload{
		Nodes:         insights.ConvertToRequestProductsDownloadNode(allProducts),
		GlobalCatalog: true,
	}
	err = cl.Parameters.ProductsDownload(rpd, fp)
	if err != nil {
		log.Fatalf("Error ProductsDownload:%v", err)
	}

	log.Printf("download %s", fname)
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

	if cfg.StartDate == "" || cfg.FinishDate == "" {
		cfg.FinishDate = time.Now().UTC().Add(-4 * 24 * time.Hour).Truncate(24 * time.Hour).Format(time.RFC3339)
		cfg.StartDate = time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
	}

	return cfg
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
