package api_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	grpc "google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	errcatapi "github.com/agschwender/errcat-go/api"
	pb "github.com/agschwender/errcat-go/protos/api"
	pbmocks "github.com/agschwender/errcat-go/protos/api/mocks"
)

type ClientTestSuite struct {
	suite.Suite

	api    *pbmocks.MockAPIClient
	client errcatapi.Client
	ctrl   *gomock.Controller
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.api = pbmocks.NewMockAPIClient(s.ctrl)
	s.client = errcatapi.NewClientWithDependencies(s.api)
}

func (s *ClientTestSuite) TearDownTest() {
	s.client.Close()
}

func (s *ClientTestSuite) TestRecordCalls() {
	ctx := context.TODO()

	req := errcatapi.RecordCallsRequest{
		Calls: []errcatapi.Call{
			{
				Dependency: "mysql",
				Duration:   time.Duration(60) * time.Second,
				Error:      errors.New("oops"),
				Name:       "orders.Purchase",
				StartedAt:  time.Now(),
			},
			{
				Dependency: "google",
				Duration:   time.Duration(120) * time.Second,
				Error:      nil,
				Name:       "google.Search",
				StartedAt:  time.Now(),
			},
		},
		Environment: "dev",
	}

	s.api.EXPECT().
		RecordCalls(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, protoReq *pb.RecordCallsRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			s.assertRecordCallsRequest(req, protoReq)
			return &emptypb.Empty{}, nil
		})

	err := s.client.RecordCalls(ctx, req)
	s.Require().NoError(err)
}

func (s *ClientTestSuite) TestRecordCallsWithError() {
	req := errcatapi.RecordCallsRequest{
		Calls:       []errcatapi.Call{{}},
		Environment: "dev",
	}

	s.api.EXPECT().
		RecordCalls(gomock.Any(), gomock.Any()).
		Return(&emptypb.Empty{}, fmt.Errorf("oops"))

	err := s.client.RecordCalls(context.TODO(), req)
	s.Require().Error(err)
	s.Equal("oops", err.Error())
}

func (s *ClientTestSuite) TestRecordCallsWithoutCalls() {
	req := errcatapi.RecordCallsRequest{Environment: "dev"}
	err := s.client.RecordCalls(context.TODO(), req)
	s.Require().NoError(err)
}

func (s *ClientTestSuite) assertRecordCallsRequest(req errcatapi.RecordCallsRequest, protoReq *pb.RecordCallsRequest) {
	s.Equal(req.Environment, protoReq.GetEnv())
	s.Require().Len(protoReq.GetCalls(), len(req.Calls))
	for i, protoCall := range protoReq.GetCalls() {
		s.assertCall(req.Calls[i], protoCall)
	}
}

func (s *ClientTestSuite) assertCall(call errcatapi.Call, protoCall *pb.Call) {
	s.Equal(call.Dependency, protoCall.GetDependency())
	s.Equal(call.Duration, protoCall.GetDuration().AsDuration())
	if call.Error == nil {
		s.Equal("", protoCall.GetError())
	} else {
		s.Equal(call.Error.Error(), protoCall.GetError())
	}
	s.Equal(call.Name, protoCall.GetName())
	s.Equal(call.StartedAt.UTC(), protoCall.GetStartedAt().AsTime().UTC())
}
