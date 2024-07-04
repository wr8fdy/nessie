// Package main implements a test client that starts a scan, wait until it finishes and exports its results to a csv file.
package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/wr8fdy/nessie"
)

var apiURL, username, password, fingerprints string

func init() {
	flag.StringVar(&apiURL, "api_url", "", "")
	flag.StringVar(&username, "username", "", "Username to login with, in production read that from a file, do not set from the command line or it will end up in your history.")
	flag.StringVar(&password, "password", "", "Password that matches the provided username, in production read that from a file, do not set from the command line or it will end up in your history.")
	flag.StringVar(&fingerprints, "fingerprints", "", "Comma-separated list of SPKI Fingerprints for the Nessus server using SHA-256 encoded in base64.")
	flag.Parse()
}

func main() {
	var err error
	var nessus nessie.Nessus
	if len(fingerprints) > 0 {
		nessus, err = nessie.NewFingerprintedNessus(apiURL, strings.Split(fingerprints, ","))
	} else {
		nessus, err = nessie.NewInsecureNessus(apiURL)
	}
	if err != nil {
		panic(err)
	}

	if err := nessus.Login(username, password); err != nil {
		log.Println(err)
		return
	}
	log.Println("Logged-in")
	defer nessus.Logout()

	var scanID int64 = 13
	var templateID int64 = 1
	// We only care about the last scan, so no use for the scan UUID here.
	if _, err = nessus.StartScan(scanID); err != nil {
		panic(err)
	}
	for {
		details, err := nessus.ScanDetails(scanID)
		if err != nil {
			panic(err)
		}
		if strings.ToLower(details.Info.Status) == "completed" {
			log.Println("Scan completed")
			break
		}
		log.Println("Scan is", details.Info.Status)
		time.Sleep(5 * time.Second)
	}

	exportID, err := nessus.ExportScan(scanID, templateID, nessie.ExportCSV)
	if err != nil {
		panic(err)
	}
	for {
		if finished, err := nessus.ExportFinished(scanID, exportID); err != nil {
			panic(err)
		} else if finished {
			log.Println("Scan export finished")
			break
		}
		log.Println("Scan export ongoing...")
		time.Sleep(5 * time.Second)
	}
	csv, err := nessus.DownloadExport(scanID, exportID)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("report.csv", csv, 0600); err != nil {
		panic(err)
	}
}
