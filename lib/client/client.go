package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	pb "github.com/thompsy/go-linux-worker/lib/protobuf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client is a gRPC client that can connect to the worker-api and execute commands.
type Client struct {
	conn   *grpc.ClientConn
	client pb.WorkerServiceClient
}

// Submit sends the given command to the server and returns the id of the resulting job.
func (c *Client) Submit(cmd string) (string, error) {
	response, err := c.client.Submit(context.Background(), &pb.Command{
		Command: cmd,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start command %s: %w", cmd, err)
	}
	return response.Id, nil
}

// Stop cancels the job identified by the given jobID.
func (c *Client) Stop(jobID string) error {
	req := &pb.JobId{
		Id: jobID,
	}
	_, err := c.client.Stop(context.Background(), req)
	if err != nil {
		return err
	}
	log.Infof("%s: killed", jobID)
	return nil
}

// Status returns the status of the job identified by the given jobID.
func (c *Client) Status(jobID string) (*pb.StatusResponse, error) {
	req := &pb.JobId{
		Id: jobID,
	}
	resp, err := c.client.Status(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get status for id %s: %w", jobID, err)
	}

	return resp, nil
}

// GetLogs fetches the logs from the server and writes them to an io.Pipe.
// The io.PipeReader is returned to the client for consumption.
func (c *Client) GetLogs(jobID string) (io.Reader, error) {
	req := &pb.JobId{
		Id: jobID,
	}

	stream, err := c.client.GetLogs(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for id: %s: %w", jobID, err)
	}

	reader, writer := io.Pipe()

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				_ = writer.Close()
				return
			}
			if err != nil {
				_ = writer.CloseWithError(err)
				return
			}
			line := resp.GetLogLine()
			_, err = writer.Write([]byte(line + "\n"))
			if err != nil {
				_ = writer.CloseWithError(err)
				return
			}
		}
	}()
	return reader, nil
}

// Close closes the connection to the server.
func (c *Client) Close() error {
	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	return nil
}

// NewClient constructs a new client with the given configuration.
func NewClient(address, caCertFile, clientCertFile, clientKeyFile string) (*Client, error) {

	// Load certificate of the CA who signed server's certificate
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add server to CA's cert: %w", err)
	}

	// Load client's certificate and private key
	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert: %w", err)
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
	}

	//todo: think about timeouts again
	// ctx, err := context.WithTimeout(context.Background(), 5 * time.Second())
	// if err != nil {
	// 	log.WithError(err).Fatal("failed to create context")
	// 	return nil, err
	// }

	conn, err := grpc.DialContext(context.Background(), address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial server: %w", err)
	}

	//todo I probably need to add the VerifyHostname method on conn. Also do this on the server.
	client := pb.NewWorkerServiceClient(conn)

	c := Client{
		conn:   conn,
		client: client,
	}
	return &c, nil
}
