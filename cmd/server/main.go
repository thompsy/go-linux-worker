/*
Package main extracts any relevant config and runs the server implemented by the library.

NOTE: the config below would typically be read in from a config file or using a command line parser
like Kong or Viper, but hard-coding things should be sufficient for this exercise.
*/
package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/thompsy/worker-api-service/lib/server"
)

func main() {
	conf := server.Config{
		CaCertFile:     "./certs/ca.crt",
		ServerCertFile: "./certs/server.crt",
		ServerKeyFile:  "./certs/server.key",
		Address:        ":8080",
	}

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
