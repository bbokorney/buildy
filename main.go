package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/DisposaBoy/JsonConfigReader"
)

var (
	logfile    = flag.String("log", "", "the name of the log file")
	configfile = flag.String("config", "buildy.config",
		"the name of the config file")
)

type Config struct{}

func main() {
	// parse flags
	flag.Parse()
	// open log file
	if *logfile != "" {
		lfile, err := os.Create(*logfile)
		defer lfile.Close()
		if err == nil {
			log.SetOutput(lfile)
		} else {
			log.Println("Cannot open logfile, logging to stderr")
		}
	} else {
		log.Println("No log file specified, logging to stderr")
	}

	// open config file
	log.Printf("Opening config file %v\n", *configfile)
	var config interface{}
	cfile, err := os.Open(*configfile)
	defer cfile.Close()
	if err == nil {
		r := JsonConfigReader.New(cfile)
		err := json.NewDecoder(r).Decode(&config)
		if err != nil {
			log.Fatalf("Could not decode the config file: %v\n", err)
		}
		log.Println(config)
	} else {
		log.Fatalf("Could not open config file: %v\n", err)
	}

}
