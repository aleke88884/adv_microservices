package client

import (
	"context"
	doctorpb "doctor-service/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type DoctorClient interface {
	DoctorExists(doctorID string) (bool, error)
}

type GRPCDoctorClient struct {
	client doctorpb.DoctorServiceClient
}

func NewGRPCDoctorClient(addr string) (*GRPCDoctorClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCDoctorClient{
		client: doctorpb.NewDoctorServiceClient(conn),
	}, nil
}

func (c *GRPCDoctorClient) DoctorExists(doctorID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.client.GetDoctor(ctx, &doctorpb.GetDoctorRequest{Id: doctorID})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.NotFound {
				return false, nil
			}
			if st.Code() == codes.Unavailable {
				return false, err
			}
		}
		return false, err
	}
	return true, nil
}
