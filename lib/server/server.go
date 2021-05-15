package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/thompsy/go-linux-worker/lib/backend"
	pb "github.com/thompsy/go-linux-worker/lib/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Config contains the configuration options required by the Server
type Config struct {
	CaCertFile     string
	ServerCertFile string
	ServerKeyFile  string
	Address        string
}

// Server is a gRPC server which implements the worker-api.
type Server struct {
	pb.UnimplementedWorkerServiceServer
	grpc *grpc.Server
	*Config
	worker *backend.Worker
}

// Submit passes the command to the worker library and returns the JobId of the resulting process.
func (s Server) Submit(ctx context.Context, in *pb.Command) (*pb.JobId, error) {
	jobId, err := s.worker.Submit(in.Command)
	if err != nil {
		return nil, fmt.Errorf("failed to start command %s: %w", in.Command, err)
	}
	return &pb.JobId{
		Id: jobId.String(),
	}, nil
}

// Stop aborts the job identified by the given JobId.
func (s Server) Stop(ctx context.Context, in *pb.JobId) (*pb.Empty, error) {
	jobID, err := uuid.FromString(in.Id)
	if err != nil {
		return nil, err
	}
	err = s.worker.Stop(jobID)
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// Status returns the status of the job identified by the given JobId.
func (s Server) Status(ctx context.Context, in *pb.JobId) (*pb.StatusResponse, error) {
	jobID, err := uuid.FromString(in.Id)
	if err != nil {
		return nil, err
	}
	status, err := s.worker.Status(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status for jobId: %s: %w", in.Id, err)
	}

	return &pb.StatusResponse{
		Status:   pb.StatusResponse_StatusType(status.Status),
		ExitCode: int32(status.ExitCode),
	}, nil
}

// GetLogs returns a stream of logs from the given JobId.
func (s Server) GetLogs(in *pb.JobId, stream pb.WorkerService_GetLogsServer) error {
	jobID, err := uuid.FromString(in.Id)
	if err != nil {
		return err
	}
	reader, err := s.worker.Logs(stream.Context(), jobID)
	if err != nil {
		return fmt.Errorf("unable to get logs for jobId %s: %w", in.Id, err)
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		t := scanner.Text()
		err := stream.Send(&pb.Log{LogLine: t})
		if err != nil {
			return fmt.Errorf("unable to stream logs for jobId %s: %w", in.Id, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to fetch logs for jobId %s: %w", in.Id, err)
	}
	return nil
}

// Serve starts the server.
func (s Server) Serve() error {
	log.Info("Starting to serve...")
	l, err := net.Listen("tcp", s.Config.Address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	err = s.grpc.Serve(l)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// Close stops the server and closes any connections.
func (s Server) Close() {
	s.grpc.Stop()
}

// NewServer constructs a server from the given configuration.
func NewServer(c Config) (*Server, error) {
	// Load our trusted CA certificate
	caCert, err := ioutil.ReadFile(c.CaCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA to pool: %w", err)
	}

	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair(c.ServerCertFile, c.ServerKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server cert: %w", err)
	}

	config := &tls.Config{
		Certificates:             []tls.Certificate{serverCert},
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                caPool,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS13,
	}

	timeout, err := time.ParseDuration("5s")
	if err != nil {
		return nil, fmt.Errorf("unable to parse timeout duration: %w", err)
	}
	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(config)),
		grpc.UnaryInterceptor(unaryAuthorizationInterceptor),
		grpc.StreamInterceptor(authorizationStreamInterceptor()),
		grpc.ConnectionTimeout(timeout),
	)

	w := Server{
		Config: &c,
		grpc:   s,
		worker: backend.NewWorker(),
	}
	pb.RegisterWorkerServiceServer(s, w)
	return &w, nil
}
