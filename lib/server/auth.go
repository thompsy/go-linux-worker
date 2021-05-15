package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/thompsy/go-linux-worker/lib"
	pb "github.com/thompsy/go-linux-worker/lib/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

var jobs map[string]*string = map[string]*string{}
var lock sync.RWMutex

//todo we could add a group() function
// isAuthorized returns true if the given clientID is authorized to access the jobID.
func isAuthorized(clientID *string, jobID string) bool {
	if *clientID == "admin@example.com" {
		return true
	}
	lock.RLock()
	owningClientID, ok := jobs[jobID]
	lock.RUnlock()
	if !ok {
		return false
	}
	return *clientID == *owningClientID
}

// getClientCertSerialNumber extracts the TLS certificate number from client certificate.
func clientIdentity(ctx context.Context) (*string, error) {
	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract peerInfo from context")
	}

	//todo think about these error messages. Do we need to return this level of detail to the client?
	authInfo, ok := peerInfo.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, fmt.Errorf("failed to extract authInfo from peerInfo")
	}

	if len(authInfo.State.PeerCertificates) != 1 {
		return nil, fmt.Errorf("unexpected number of client certificates: %d", len(authInfo.State.PeerCertificates))
	}
	return &authInfo.State.PeerCertificates[0].Subject.CommonName, nil
}

// unaryAuthorizationInterceptor intercepts the unary calls in order to determine whether
// a client has sufficient authorization to execute the requested call.
func unaryAuthorizationInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	clientID, err := clientIdentity(ctx)
	if err != nil {
		return nil, lib.ErrNotFound
	}

	// we assume that if a client has valid TLS key then they can submit jobs.
	if info.FullMethod == "/protobuf.WorkerService/Submit" {
		h, err := handler(ctx, req)
		if err == nil {
			jobID := h.(*pb.JobId).Id
			lock.Lock()
			jobs[jobID] = clientID
			lock.Unlock()
		}
		return h, err
	}

	jobID := req.(*pb.JobId).Id
	if !isAuthorized(clientID, jobID) {
		return nil, lib.ErrNotFound
	}
	return handler(ctx, req)
}

// authorizationStreamWrapper is a wrapper around a grpc.ServerStream which implements basic authorization
// checking before beginning to stream.
type authorizationStreamWrapper struct {
	grpc.ServerStream
}

// RecvMsg is called when a request to stream is received by the server. This calls the wrapped ServerStream.RecvMsg()
// in order to get the given JobId to check that the client is authorized.
func (l authorizationStreamWrapper) RecvMsg(m interface{}) error {
	err := l.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	jobID := m.(*pb.JobId)
	clientID, err := clientIdentity(l.ServerStream.Context())
	if err != nil {
		return lib.ErrNotFound
	}
	if !isAuthorized(clientID, jobID.Id) {
		return lib.ErrNotFound
	}
	return nil
}

// authorizationStreamInterceptor returns a stream interceptor which preforms basic authorization checking.
func authorizationStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, authorizationStreamWrapper{ss})
	}
}
