package backend

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/thompsy/worker-api-service/lib"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"
)

const (
	wcCommand = "wc -l ./test-slow-command.sh"
	wcOutput  = "12 ./test-slow-command.sh\n"

	slowCommand = "./test-slow-command.sh"
	slowOutput  = "test-command.sh :: 1\n" +
		"test-command.sh :: 2\n" +
		"test-command.sh :: 3\n" +
		"test-command.sh :: 4\n" +
		"test-command.sh :: All done\n"
)

// TestSubmitCommand verifies that the worker accepts a new command and that
// it's status and logs can be fetched.
func TestSubmitCommand(t *testing.T) {
	skipCI(t)
	w := NewWorker()
	jobID, err := w.Submit(wcCommand)
	require.Nil(t, err)

	// Sleep for a moment to allow the command to finish.
	time.Sleep(time.Second)

	status, err := w.Status(jobID)
	require.Nil(t, err)
	require.Equal(t, 0, status.ExitCode)
	require.Equal(t, lib.COMPLETED, status.Status)

	reader, err := w.Logs(context.Background(), jobID)
	require.Nil(t, err)

	output, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	require.Contains(t, string(output), wcOutput)
}

// TestStopCommand verifies that the worker can stop a command and that the
// status is reported correctly.
func TestStopCommand(t *testing.T) {
	skipCI(t)
	w := NewWorker()
	jobID, err := w.Submit(slowCommand)
	require.Nil(t, err)

	status, err := w.Status(jobID)
	require.Nil(t, err)
	require.Equal(t, 0, status.ExitCode)
	require.Equal(t, lib.RUNNING, status.Status)

	err = w.Stop(jobID)
	require.Nil(t, err)

	status, err = w.Status(jobID)
	require.Nil(t, err)
	require.Equal(t, -1, status.ExitCode)
	require.Equal(t, lib.STOPPED, status.Status)
}

// TestConcurrentRead verifies that readers can read correctly from a slow writer.
func TestConcurrentLogs(t *testing.T) {
	skipCI(t)
	w := NewWorker()
	jobID, err := w.Submit("./test-slow-command.sh")
	require.Nil(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := w.Logs(context.Background(), jobID)
		require.Nil(t, err)
		output, err := ioutil.ReadAll(r)
		require.Nil(t, err)
		require.Equal(t, slowOutput, string(output))
	}()

	time.Sleep(5 * time.Second)

	status, err := w.Status(jobID)
	require.Nil(t, err)
	require.Equal(t, 0, status.ExitCode)
	require.Equal(t, lib.COMPLETED, status.Status)

	wg.Wait()
}

func TestContextTimeout(t *testing.T) {
	skipCI(t)
	w := NewWorker()
	jobID, err := w.Submit("./test-slow-command.sh")
	require.Nil(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1*time.Second))
	defer cancel()

	go func() {
		r, err := w.Logs(ctx, jobID)
		require.Nil(t, err)
		_, _ = ioutil.ReadAll(r)
	}()

	<-ctx.Done()
	switch ctx.Err() {
	case context.DeadlineExceeded:
	case context.Canceled:
		t.Fail()
	}
}

// skipCI skips the current test if running in a CI environment. These tests
// cannot be run in a non-privileged container.
func skipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}
