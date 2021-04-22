package testing

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	c "github.com/thompsy/worker-api-service/lib/client"
	s "github.com/thompsy/worker-api-service/lib/server"
)

const address = ":8081"

func TestMain(m *testing.M) {
	server, err := setup()
	if err != nil {
		os.Exit(1)
	}

	go func(server *s.Server) {
		err := server.Serve()
		if err != nil {
			fmt.Printf("error starting test server: %s", err)
		}

	}(server)

	retCode := m.Run()

	server.Close()
	os.Exit(retCode)
}

func setup() (*s.Server, error) {
	conf := s.Config{
		CaCertFile:     "../certs/ca.crt",
		ServerCertFile: "../certs/server.crt",
		ServerKeyFile:  "../certs/server.key",
		Address:        address,
	}

	server, err := s.NewServer(conf)
	return server, err
}

func TestAuthentication(t *testing.T) {
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

func TestAuthorization(t *testing.T) {
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
