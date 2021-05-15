/*
Package main implements a simple demo client for the Worker API server.
*/
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	log "github.com/sirupsen/logrus"
	c "github.com/thompsy/go-linux-worker/lib/client"
	"github.com/thompsy/go-linux-worker/lib/protobuf"
)

// NOTE: these constants would ideally be pulled out to a config file or passed in
// using a command-line parser like Kong or Viper.
const (
	caCertFile = "./certs/ca.crt"
)

// Context contains data that is used by more than one of the commands.
type Context struct {
	Client *c.Client
}

// SubmitCmd represents the arguments needed when submitting a new command to the server.
type SubmitCmd struct {
	Command string `arg name:"command" help:"Command to run." type:"string"`
}

// Run submits the command to the server.
func (s *SubmitCmd) Run(ctx *Context) error {
	jobID, err := ctx.Client.Submit(s.Command)
	if err != nil {
		fmt.Printf("Error submitting job: %s\n", err)
		return err
	}
	fmt.Printf("Id of submitted job: %s\n", jobID)
	return nil
}

// StopCmd represents the arguments needed to stop a running job.
type StopCmd struct {
	JobID string `arg name:"jobID" help:"JobID to stop." type:"string"`
}

// Run stops the job identified by the given JobID.
func (s *StopCmd) Run(ctx *Context) error {
	err := ctx.Client.Stop(s.JobID)
	if err != nil {
		fmt.Printf("Error stopping job %s: %s\n", s.JobID, err)
		return err
	}
	return nil
}

// LogsCmd represents the arguments needed to fetch the logs for a job.
type LogsCmd struct {
	JobID string `arg name:"jobID" help:"JobID to stop." type:"string"`
}

// Run fetches the logs identified by the given JobID.
func (l *LogsCmd) Run(ctx *Context) error {
	reader, err := ctx.Client.GetLogs(l.JobID)
	if err != nil {
		fmt.Printf("Error fetching logs for job %s: %s\n", l.JobID, err)
		return err
	}

	if _, err := io.Copy(os.Stdout, reader); err != nil {
		fmt.Printf("Error fetching logs for job %s: %s\n", l.JobID, err)
		return err
	}
	return nil
}

// StatusCmd represents the arguments needed to query the status of a job.
type StatusCmd struct {
	JobID string `arg name:"jobID" help:"JobID to stop." type:"string"`
}

// Run gets the status of the job identified by the given JobID.
func (s *StatusCmd) Run(ctx *Context) error {
	status, err := ctx.Client.Status(s.JobID)
	if err != nil {
		fmt.Printf("Error fetching status for job %s: %s\n", s.JobID, err)
		return err
	}

	fmt.Printf("Status: %s\n", status.Status.String())
	if status.Status == protobuf.StatusResponse_COMPLETED {
		fmt.Printf("Exit code: %d\n", status.ExitCode)
	}
	return nil
}

// cli represents the available command line options.
var cli struct {
	Submit SubmitCmd `cmd help:"Submit command."`
	Stop   StopCmd   `cmd help:"Stop the given JobID."`
	Status StatusCmd `cmd help:"Get the status of the given JobID."`
	Logs   LogsCmd   `cmd help:"Get the logs for the given JobID."`

	Profile string `short:"p" help:"TLS profile to connect with (a|b|admin)." default:"a"`
	Address string `short:"h" help:"Address of the server." default:":8080"`
}

func main() {
	ctx := kong.Parse(&cli)

	//TODO profile should probably be split out into several cli arguments.
	log.Infof("Using profile: %s", cli.Profile)
	clientCertFile := fmt.Sprintf("./certs/client_%s.crt", cli.Profile)
	clientKeyFile := fmt.Sprintf("./certs/client_%s.key", cli.Profile)
	client, err := c.NewClient(cli.Address, caCertFile, clientCertFile, clientKeyFile)
	if err != nil {
		log.WithError(err).Fatal("error creating client")
	}
	defer client.Close()

	err = ctx.Run(&Context{Client: client})
	ctx.FatalIfErrorf(err)
}
