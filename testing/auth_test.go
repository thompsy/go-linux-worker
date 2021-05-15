package testing

import (
	"github.com/stretchr/testify/require"
	c "github.com/thompsy/go-linux-worker/lib/client"
	pb "github.com/thompsy/go-linux-worker/lib/protobuf"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"
)

const address = "server:8080"

// TestMain runs the test suite.
func TestMain(m *testing.M) {
	retCode := m.Run()
	os.Exit(retCode)
}

// TestSubmitJob submits a test job, fetches the logs and verifies the status.
func TestSubmitJob(t *testing.T) {
	skipCI(t)

	client, err := c.NewClient(address, "../certs/ca.crt", "../certs/client_a.crt", "../certs/client_a.key")
	require.Nil(t, err, "failed to create client")

	jobID, err := client.Submit("whoami")
	require.Nil(t, err, "failed to submit job")

	// Sleep for a moment to allow the command to finish.
	time.Sleep(2 * time.Second)

	status, err := client.Status(jobID)
	require.Nil(t, err, "failed to get status")
	require.Equal(t, int32(0), status.ExitCode)
	require.Equal(t, pb.StatusResponse_COMPLETED, status.Status)

	reader, err := client.GetLogs(jobID)
	require.Nil(t, err)

	output, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, string(output), "root\n")
}

// TestStopJob verifies that a submitted job can be successfully stopped.
func TestStopJob(t *testing.T) {
	skipCI(t)

	client, err := c.NewClient(address, "../certs/ca.crt", "../certs/client_a.crt", "../certs/client_a.key")
	require.Nil(t, err, "failed to create client")

	jobID, err := client.Submit("sleep 100")
	require.Nil(t, err, "failed to submit job")

	status, err := client.Status(jobID)
	require.Nil(t, err, "failed to get status")
	require.Equal(t, int32(0), status.ExitCode)
	require.Equal(t, pb.StatusResponse_RUNNING, status.Status)

	err = client.Stop(jobID)
	require.Nil(t, err)

	status, err = client.Status(jobID)
	require.Nil(t, err)
	require.Equal(t, int32(-1), status.ExitCode)
	require.Equal(t, pb.StatusResponse_STOPPED, status.Status)
}

// TestConcurrentLogs verifies that readers can read correctly from a slow writer.
func TestConcurrentLogs(t *testing.T) {
	skipCI(t)
	client, err := c.NewClient(address, "../certs/ca.crt", "../certs/client_a.crt", "../certs/client_a.key")
	require.Nil(t, err, "failed to create client")

	jobID, err := client.Submit("find /")
	require.Nil(t, err, "failed to submit job")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := client.GetLogs(jobID)
		require.Nil(t, err)
		output, err := ioutil.ReadAll(r)
		require.Nil(t, err)
		require.Equal(t, 101424, len(output))
	}()
	wg.Wait()
}

// TestAuthentication verifies that only trusted clients can connect to the server.
func TestAuthentication(t *testing.T) {
	skipCI(t)
	tests := []struct {
		desc       string
		caCert     string
		clientCert string
		clientKey  string
		assertErr  require.ErrorAssertionFunc
	}{
		{
			desc:       "successful case",
			caCert:     "../certs/ca.crt",
			clientCert: "../certs/client_a.crt",
			clientKey:  "../certs/client_a.key",
			assertErr:  require.NoError,
		},
		{
			desc:       "server does not recognise client ca",
			caCert:     "../certs/ca.crt",
			clientCert: "../certs/untrusted_client.crt",
			clientKey:  "../certs/untrusted_client.key",
			assertErr:  require.Error,
		},
		{
			desc:       "client does not recognise server ca",
			caCert:     "../certs/untrusted_ca.crt",
			clientCert: "../certs/client_a.crt",
			clientKey:  "../certs/client_a.key",
			assertErr:  require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			client, err := c.NewClient(address, tt.caCert, tt.clientCert, tt.clientKey)
			if err != nil {
				t.Errorf("client setup failed: %s", err)
			}
			_, err = client.Submit("whoami")
			tt.assertErr(t, err)
		},
		)
	}
}

// TestAuthorization verifies that clients can view appropriate jobs.
func TestAuthorization(t *testing.T) {
	skipCI(t)
	caCert := "../certs/ca.crt"

	tests := []struct {
		desc        string
		caCert      string
		clientACert string
		clientAKey  string
		clientBCert string
		clientBKey  string
		assertErr   require.ErrorAssertionFunc
	}{
		{
			desc:        "clients with same certs can access shared jobs",
			clientACert: "../certs/client_a.crt",
			clientAKey:  "../certs/client_a.key",
			clientBCert: "../certs/client_a.crt",
			clientBKey:  "../certs/client_a.key",
			assertErr:   require.NoError,
		},
		{
			desc:        "clients with different valid certs cannot access shared jobs",
			clientACert: "../certs/client_a.crt",
			clientAKey:  "../certs/client_a.key",
			clientBCert: "../certs/client_b.crt",
			clientBKey:  "../certs/client_b.key",
			assertErr:   require.Error,
		},
		{
			desc:        "admin clients can access all jobs",
			clientACert: "../certs/client_a.crt",
			clientAKey:  "../certs/client_a.key",
			clientBCert: "../certs/client_admin.crt",
			clientBKey:  "../certs/client_admin.key",
			assertErr:   require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			clientA, err := c.NewClient(address, caCert, tt.clientACert, tt.clientAKey)
			if err != nil {
				t.Errorf("client setup failed: %s", err)
			}
			jobID, err := clientA.Submit("whoami")
			if err != nil {
				t.Errorf("client A setup failed: %s", err)
			}

			clientB, err := c.NewClient(address, caCert, tt.clientBCert, tt.clientBKey)
			if err != nil {
				t.Errorf("client B setup failed: %s", err)
			}
			_, err = clientB.Status(jobID)
			tt.assertErr(t, err)
			r, err := clientB.GetLogs(jobID)
			if err != nil {
				t.Errorf("initial call to GetLogs failed")
			}
			_, err = ioutil.ReadAll(r)
			tt.assertErr(t, err)
		},
		)
	}
}

// skipCI skips the current test if running in a CI environment. These tests
// cannot be run in a non-privileged container.
func skipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}
