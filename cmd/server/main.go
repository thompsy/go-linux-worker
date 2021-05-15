/*
Package main extracts any relevant config and runs the server implemented by the library.

NOTE: the config below would typically be read in from a config file or using a command line parser
like Kong or Viper, but hard-coding things should be sufficient for this exercise.
*/
package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/thompsy/go-linux-worker/lib/backend"
	"github.com/thompsy/go-linux-worker/lib/server"
)

func main() {
	conf := server.Config{
		CaCertFile:     "./certs/ca.crt",
		ServerCertFile: "./certs/server.crt",
		ServerKeyFile:  "./certs/server.key",
		Address:        ":8080",
		CGroups: &backend.CGroups{
			Limits: map[string]map[string]string{
				"cpu": {
					"shares":        "100",
					"cfs_quota_us":  "1000",
					"cfs_period_us": "1000"},
				"memory": {
					"limit_in_bytes": "1000g"},
				//NOTE: this does not work on Windows due to WSL issues
				// "blkio": {
				//  	"throttle.read_bps_device":  "8:0 10240",
				//  	"throttle.write_bps_device": "8:0 10240"},
			},
		},
	}

	// If run with the "exec" argument just run the passed command in an isolated environment and exit.
	if len(os.Args) > 1 && os.Args[1] == "exec" {
		backend.Exec(os.Args[2])
		os.Exit(0)
	}

	// If no arguments are supplied simply start the server.
	log.Infof("Starting server. pid: %d", os.Getpid())
	s, err := server.NewServer(conf)
	if err != nil {
		log.WithError(err).Fatal("error creating server")
		os.Exit(1)
	}

	err = s.Serve()
	if err != nil {
		log.WithError(err).Fatal("error starting server")
		os.Exit(1)
	}
}
