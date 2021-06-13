package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/germangorelkin/go-x5/logistics"
)

func main() {
	instance := os.Getenv("INSTANCE")
	login := os.Getenv("LOGIN")
	password := os.Getenv("PASSWORD")
	verbose, _ := strconv.ParseBool(os.Getenv("VERBOSE"))
	salesChannel := logistics.SalesChannel(os.Getenv("SALES_CHANNEL"))
	typeReport := logistics.TypeReport(os.Getenv("TYPE_REPORT"))
	startDate := os.Getenv("START_DATE")
	finishDAte := os.Getenv("FINISH_DATE")

	cli, err := logistics.NewClient(logistics.ClintConf{
		Instance: instance,
		Verbose:  verbose,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("build new client")

	token, err := cli.Auth.Auth(login, password)
	if err != nil {
		log.Fatal(err)
	}
	cli.SetToken(token)
	log.Printf("get new token:%s", token)

	// StartDate:    time.Now().UTC().Add(-1 * 24 * time.Hour).Truncate(24 * time.Hour).Format(time.RFC3339),
	// FinishDate:   time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339),
	reqCR := logistics.RequestCreateReport{
		StartDate:    startDate,
		FinishDate:   finishDAte,
		SalesChannel: salesChannel,
		TypeReport:   typeReport,
	}
	log.Printf("request of create report:%+v", reqCR)

	reportId, err := cli.Reports.Create(reqCR)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("new report created:%s", reportId)

	delay := 5 * time.Second
	var resStatus logistics.ResponseStatusReport

	for attempts := 10; attempts >= 0; attempts-- {
		time.Sleep(delay)

		resStatus, err = cli.Reports.Status(reportId)
		if err != nil {
			log.Fatal(err)
		}
		if resStatus.Result.ReportStatus == logistics.DONE ||
			resStatus.Result.ReportStatus == logistics.ERROR {
			break
		}
		log.Printf("attempt %d, status:#%+v", attempts, resStatus)
	}

	if resStatus.Result.ReportStatus != logistics.DONE {
		log.Fatalf("#%+v", resStatus)
	}
	log.Printf("status:#%+v", resStatus)

	for _, partIds := range resStatus.Result.PartIds {
		f, err := os.Create(partIds)
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
