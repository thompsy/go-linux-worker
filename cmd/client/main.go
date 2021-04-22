/*
Package main implements a simple demo client for the Worker API server.
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	c "github.com/thompsy/worker-api-service/lib/client"
	"github.com/thompsy/worker-api-service/lib/protobuf"
)

// NOTE: these constants would ideally be pulled out to a config file or passed in
// using a command-line parser like Kong or Viper.
const (
	caCertFile = "./certs/ca.crt"
	address    = ":8080"
)

func main() {
	var status string
	var stop string
	var logs string
	var profile string

	// NOTE: more sophisticated command line parsing like Kong could be used here
	// but I've stuck with the flag package for simplicity.
	flag.StringVar(&status, "s", "", "Job Id to fetch status for")
	flag.StringVar(&stop, "k", "", "Job Id to stop")
	flag.StringVar(&logs, "l", "", "Job Id to fetch logs for")
	flag.StringVar(&profile, "p", "a", "Client identity to use. Valid values: a, b, admin")
	flag.Parse()

	command := strings.Join(flag.Args()[:], " ")

	clientCertFile := fmt.Sprintf("./certs/client_%s.crt", profile)
	clientKeyFile := fmt.Sprintf("./certs/client_%s.key", profile)
	client, err := c.NewClient(address, caCertFile, clientCertFile, clientKeyFile)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer client.Close()

	if command != "" {
		jobID, err := client.Submit(command)
		if err != nil {
			fmt.Printf("Error submitting job: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Id of submitted job: %s\n", jobID)
		return

	} else if stop != "" {
		err := client.Stop(stop)
		if err != nil {
			fmt.Printf("Error stopping job %s: %s\n", stop, err)
			os.Exit(1)
		}
		return

	} else if status != "" {
		s, err := client.Status(status)
		if err != nil {
			fmt.Printf("Error fetching status for job %s: %s\n", status, err)
			os.Exit(1)
		}

		fmt.Printf("Status: %s\n", s.Status.String())
		if s.Status == protobuf.StatusResponse_COMPLETED {
			fmt.Printf("Exit code: %d\n", s.ExitCode)
		}
		return

	} else if logs != "" {
		reader, err := client.GetLogs(logs)
		if err != nil {
			fmt.Printf("Error fetching logs for job %s: %s\n", logs, err)
			os.Exit(1)
		}

		if _, err := io.Copy(os.Stdout, reader); err != nil {
			fmt.Printf("Error fetching logs for job %s: %s\n", logs, err)
			os.Exit(1)
		}
		return
	}
}
