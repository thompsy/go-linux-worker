package lib

import "errors"

var (
	// ErrNotFound is the standard error which will be returned if we are unable to authorize the client for any reason.
	ErrNotFound = errors.New("job not found")
)

// Status provides status information about a client submitted job.
type Status struct {
	Status StatusCode

	// ExitCode is the exit code returned by the command. It's value is
	// only meaningful if the Status is COMPLETED or STOPPED.
	ExitCode int
}

// StatusCode is an int type that represents whether a job is running,
// completed or has been stopped.
type StatusCode int

const (
	RUNNING StatusCode = iota
	COMPLETED
	STOPPED
)
