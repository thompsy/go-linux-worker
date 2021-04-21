package backend

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	//"syscall"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/thompsy/worker-api-service/lib"
)

// A Worker is a map guarded by a RWMutex which contains an entry for each
// successfully started job.
type Worker struct {
	jobs map[uuid.UUID]*job
	sync.RWMutex
}

// A job is an exec.Cmd and its associated status and output reader.
type job struct {
	cmd *exec.Cmd

	// status is the current status of the job.
	status    lib.Status
	statusMtx sync.RWMutex

	// output contains the stdout and stderr from the command.
	output *broadcastBuffer

	// stopped is closed once the cmd has been successfully stopped
	// after a call to Stop(). This prevents the Stop() method from
	// returning before the actual cmd has been stopped.
	stopped chan struct{}
}

// NewWorker returns a correctly initialized worker struct.
func NewWorker() *Worker {
	return &Worker{
		jobs: make(map[uuid.UUID]*job),
	}
}

// Submit runs the given command in a goroutine and returns the ID of the job.
func (w *Worker) Submit(cmdLine string) (uuid.UUID, error) {
	cmdParts := strings.Split(cmdLine, " ")
	if len(cmdParts) < 1 {
		return uuid.Nil, fmt.Errorf("no command supplied")
	}

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	buffer := newBroadcastBuffer()
	cmd.Stdout = buffer
	cmd.Stderr = buffer
	//cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}

	j := &job{
		cmd:     cmd,
		output:  buffer,
		stopped: make(chan struct{}, 1),
	}

	err := cmd.Start()
	if err != nil {
		log.WithError(err).Errorf("failed to start job: %s", cmdLine)
		return uuid.Nil, err
	}
	j.status = lib.Status{Status: lib.RUNNING}
	jobID := uuid.NewV4()
	w.Lock()
	w.jobs[jobID] = j
	w.Unlock()
	log.WithField("jobID", jobID).Infof("started command: %s", cmdLine)

	// this goroutine waits for command to complete before updating the
	// status and closing the output buffer
	go func() {
		err := cmd.Wait()

		j.statusMtx.Lock()
		if err != nil && err.Error() == "signal: killed" {
			j.status = lib.Status{
				Status:   lib.STOPPED,
				ExitCode: j.cmd.ProcessState.ExitCode(),
			}
			log.WithField("jobID", jobID).Info("job stopped")
		} else {
			j.status = lib.Status{
				Status:   lib.COMPLETED,
				ExitCode: j.cmd.ProcessState.ExitCode(),
			}
			log.WithField("jobID", jobID).Info("job complete")
		}
		j.statusMtx.Unlock()
		close(j.stopped)
		buffer.Close()
	}()

	return jobID, nil
}

// Stop kills the job identified by jobID.
func (w *Worker) Stop(jobID uuid.UUID) error {
	job, err := w.getJob(jobID)
	if err != nil {
		return err
	}

	err = job.cmd.Process.Kill()
	if err != nil {
		log.WithError(err).WithField("jobID", jobID).Error("failed to stop job")
		return err
	}

	// Wait until the channel has been closed so that we know that the
	// underlying process has indeed been stopped.
	<-job.stopped

	return nil
}

// Status returns the status of the job identified by jobID.
func (w *Worker) Status(jobID uuid.UUID) (lib.Status, error) {
	job, err := w.getJob(jobID)
	if err != nil {
		return lib.Status{}, err
	}

	job.statusMtx.RLock()
	defer job.statusMtx.RUnlock()
	return job.status, nil
}

// Logs returns an io.Reader attached to both the stdout and stderr of the
// job identified by jobID.
func (w *Worker) Logs(ctx context.Context, jobID uuid.UUID) (io.Reader, error) {
	job, err := w.getJob(jobID)
	if err != nil {
		return nil, err
	}

	return job.output.NewReader(ctx), nil
}

// getJob returns the *job identified by jobID or ErrUnknownJob
func (w *Worker) getJob(jobID uuid.UUID) (*job, error) {
	w.RLock()
	job, ok := w.jobs[jobID]
	w.RUnlock()
	if !ok {
		return nil, lib.ErrNotFound
	}
	return job, nil
}
