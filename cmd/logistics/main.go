package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/germangorelkin/go-x5/logistics"
)

func main() {
	cfg := config()

	cli, err := logistics.NewClient(logistics.ClintConf{
		Instance: cfg.instance,
		Login:    cfg.login,
		Password: cfg.password,
		Verbose:  cfg.verbose,
		AutoAuth: cfg.autoAuth,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("build new client")

	// token, err := cli.Auth.Auth(cfg.login, cfg.password)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// cli.SetToken(token)
	// log.Printf("get new token:%s", token)

	reqCR := logistics.RequestCreateReport{
		StartDate:    cfg.startDate,
		FinishDate:   cfg.finishDAte,
		SalesChannel: cfg.salesChannel,
		TypeReport:   cfg.typeReport,
		IsArchive:    cfg.isArchive,
	}
	log.Printf("request of create report:%+v", reqCR)

	reportId, err := cli.Reports.Create(reqCR)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("new report created:%s", reportId)

	delay := time.Duration(cfg.waiteReportStatusDelaySec) * time.Second
	var resStatus logistics.ResponseStatusReport

	for attempts := cfg.waiteReportStatusAttempt; attempts >= 0; attempts-- {
		log.Printf("wait %dsec report status", cfg.waiteReportStatusDelaySec)
		time.Sleep(delay)

		resStatus, err = cli.Reports.Status(reportId)
		if err != nil {
			log.Fatal(err)
		}
		if resStatus.Result.ReportStatus == logistics.DONE ||
			resStatus.Result.ReportStatus == logistics.ERROR {
			break
		}
		log.Printf("attempt %d in %d; status:#%+v", cfg.waiteReportStatusAttempt-attempts+1, cfg.waiteReportStatusAttempt, resStatus)
	}

	if resStatus.Result.ReportStatus != logistics.DONE {
		log.Fatalf("#%+v", resStatus)
	}
	log.Printf("status:#%+v", resStatus)

	for _, partIds := range resStatus.Result.PartIds {
		path := filepath.Join(cfg.outDir, partIds)
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		if err := cli.Reports.Download(partIds, f); err != nil {
			log.Println(err)
		}
		f.Close()

		log.Printf("download %s", partIds)
	}
}

func config() mainConfig {
	var cfg mainConfig

	cfg.instance = os.Getenv("INSTANCE")
	cfg.login = os.Getenv("LOGIN")
	cfg.password = os.Getenv("PASSWORD")
	cfg.verbose, _ = strconv.ParseBool(os.Getenv("VERBOSE"))
	cfg.autoAuth, _ = strconv.ParseBool(os.Getenv("AUTO_AUTH"))
	cfg.salesChannel = logistics.SalesChannel(os.Getenv("SALES_CHANNEL"))
	cfg.typeReport = logistics.TypeReport(os.Getenv("TYPE_REPORT"))
	cfg.startDate = os.Getenv("START_DATE")
	cfg.finishDAte = os.Getenv("FINISH_DATE")
	cfg.isArchive, _ = strconv.ParseBool(os.Getenv("ARCHIVE"))
	cfg.outDir = os.Getenv("OUT_DIR")
	waiteReportStatusDelaySec := os.Getenv("WAITE_REPORT_STATUS_DELAY_SEC")
	waiteReportStatusAttempt := os.Getenv("WAITE_REPORT_STATUS_ATTEMPT")

	if waiteReportStatusDelaySec == "" {
		waiteReportStatusDelaySec = "10"
	}
	n, err := strconv.Atoi(waiteReportStatusDelaySec)
	if err != nil {
		log.Fatalf("failed to parse WAITE_REPORT_STATUS_DELAY_SEC=%s", waiteReportStatusDelaySec)
	}
	cfg.waiteReportStatusDelaySec = n

	if waiteReportStatusAttempt == "" {
		waiteReportStatusAttempt = "10"
	}
	n, err = strconv.Atoi(waiteReportStatusAttempt)
	if err != nil {
		log.Fatalf("failed to parse WAITE_REPORT_STATUS_ATTEMPT=%s", waiteReportStatusAttempt)
	}
	cfg.waiteReportStatusAttempt = n

	if cfg.outDir == "" {
		cfg.outDir = "reports"
	}
	if err := os.MkdirAll(cfg.outDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create out dir %s:%v", cfg.outDir, err)
	}

	if cfg.startDate == "" || cfg.finishDAte == "" {
		cfg.finishDAte = time.Now().UTC().Add(-4 * 24 * time.Hour).Truncate(24 * time.Hour).Format(time.DateOnly)
		cfg.startDate = time.Now().UTC().Truncate(24 * time.Hour).Format(time.DateOnly)
	}

	return cfg
}

type mainConfig struct {
	instance                  string
	login                     string
	password                  string
	verbose                   bool
	autoAuth                  bool
	salesChannel              logistics.SalesChannel
	typeReport                logistics.TypeReport
	startDate                 string // Если не заполнять поле то по умолчанию указывается текущая дата.
	finishDAte                string // Если не заполнять поле то по умолчанию указывается текущая дата -4 день.
	outDir                    string //  Если не заполнять поле то по умолчанию указывается report
	waiteReportStatusDelaySec int    //  Если не заполнять поле то по умолчанию указывается 10 sec
	waiteReportStatusAttempt  int    //  Если не заполнять поле то по умолчанию указывается 10
	isArchive                 bool
}
