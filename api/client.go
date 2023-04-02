package api

import (
	"context"
	"net/url"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/agschwender/errcat-go/protos/api"
)

// Client defines the interface for interacting with the errcat-server.
type Client interface {
	Close() error
	RecordCalls(context.Context, RecordCallsRequest) error
}

// Ensure the implementation matches the interface.
var _ Client = (*client)(nil)

type client struct {
	api  pb.APIClient
	conn *grpc.ClientConn
}

// NewClient creates a new client that connects to the errcat-server.
func NewClient(addr url.URL) (Client, error) {
	conn, err := grpc.Dial(addr.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &client{
		api:  pb.NewAPIClient(conn),
		conn: conn,
	}, nil
}

// NewClientWithDependencies allows the passing of the API client
// directly. This is useful for mocking. In the event that API client is
// not a mock, the caller will be responsible for closing any underlying
// connections themselves.
func NewClientWithDependencies(api pb.APIClient) Client {
	return &client{api: api}
}

// Closes the connection to the errcat-server.
func (c *client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

type RecordCallsRequest struct {
	Calls       []Call
	Environment string
	Service     string
}

func (r RecordCallsRequest) toProto() *pb.RecordCallsRequest {
	protoCalls := make([]*pb.Call, len(r.Calls))
	for i, call := range r.Calls {
		protoCalls[i] = call.toProto()
	}

	return &pb.RecordCallsRequest{
		Calls:   protoCalls,
		Env:     r.Environment,
		Service: r.Service,
	}
}

type Call struct {
	Dependency string
	Duration   time.Duration
	Error      error
	Name       string
	StartedAt  time.Time
}

func (c Call) toProto() *pb.Call {
	var err string
	if c.Error != nil {
		err = c.Error.Error()
	}

	return &pb.Call{
		Dependency: c.Dependency,
		Duration:   durationpb.New(c.Duration),
		Error:      err,
		Name:       c.Name,
		StartedAt:  timestamppb.New(c.StartedAt),
	}
}

func (c *client) RecordCalls(ctx context.Context, req RecordCallsRequest) error {
	if c == nil || len(req.Calls) == 0 {
		return nil
	}
	_, err := c.api.RecordCalls(ctx, req.toProto())
	return err
}
