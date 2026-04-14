package grpc

import (
	"context"
	"doctor-service/internal/usecase"
	pb "doctor-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DoctorServer struct {
	pb.UnimplementedDoctorServiceServer
	useCase *usecase.DoctorUseCase
}

func NewDoctorServer(useCase *usecase.DoctorUseCase) *DoctorServer {
	return &DoctorServer{useCase: useCase}
}

func (s *DoctorServer) CreateDoctor(ctx context.Context, req *pb.CreateDoctorRequest) (*pb.DoctorResponse, error) {
	if req.FullName == "" {
		return nil, status.Error(codes.InvalidArgument, "full_name is required")
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	doctor, err := s.useCase.CreateDoctor(req.FullName, req.Specialization, req.Email)
	if err != nil {
		if err.Error() == "email must be unique" {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.DoctorResponse{
		Id:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}, nil
}

func (s *DoctorServer) GetDoctor(ctx context.Context, req *pb.GetDoctorRequest) (*pb.DoctorResponse, error) {
	doctor, err := s.useCase.GetDoctorByID(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "doctor not found")
	}

	return &pb.DoctorResponse{
		Id:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}, nil
}

func (s *DoctorServer) ListDoctors(ctx context.Context, req *pb.ListDoctorsRequest) (*pb.ListDoctorsResponse, error) {
	doctors, err := s.useCase.GetAllDoctors()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &pb.ListDoctorsResponse{}
	for _, d := range doctors {
		resp.Doctors = append(resp.Doctors, &pb.DoctorResponse{
			Id:             d.ID,
			FullName:       d.FullName,
			Specialization: d.Specialization,
			Email:          d.Email,
		})
	}
	return resp, nil
}
