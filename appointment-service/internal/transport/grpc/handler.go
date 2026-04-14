package grpc

import (
	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	pb "appointment-service/proto"
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppointmentServer struct {
	pb.UnimplementedAppointmentServiceServer
	useCase *usecase.AppointmentUseCase
}

func NewAppointmentServer(useCase *usecase.AppointmentUseCase) *AppointmentServer {
	return &AppointmentServer{useCase: useCase}
}

func (s *AppointmentServer) CreateAppointment(ctx context.Context, req *pb.CreateAppointmentRequest) (*pb.AppointmentResponse, error) {
	appointment, err := s.useCase.CreateAppointment(req.Title, req.Description, req.DoctorId)
	if err != nil {
		return nil, mapCreateError(err)
	}
	return appointmentToProto(appointment), nil
}

func (s *AppointmentServer) GetAppointment(ctx context.Context, req *pb.GetAppointmentRequest) (*pb.AppointmentResponse, error) {
	appointment, err := s.useCase.GetAppointmentByID(req.Id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "appointment not found")
	}
	return appointmentToProto(appointment), nil
}

func (s *AppointmentServer) ListAppointments(ctx context.Context, req *pb.ListAppointmentsRequest) (*pb.ListAppointmentsResponse, error) {
	appointments, err := s.useCase.GetAllAppointments()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &pb.ListAppointmentsResponse{}
	for _, a := range appointments {
		resp.Appointments = append(resp.Appointments, appointmentToProto(a))
	}
	return resp, nil
}

func (s *AppointmentServer) UpdateAppointmentStatus(ctx context.Context, req *pb.UpdateStatusRequest) (*pb.AppointmentResponse, error) {
	appointment, err := s.useCase.UpdateAppointmentStatus(req.Id, model.Status(req.Status))
	if err != nil {
		return nil, mapUpdateError(err)
	}
	return appointmentToProto(appointment), nil
}

func appointmentToProto(a *model.Appointment) *pb.AppointmentResponse {
	return &pb.AppointmentResponse{
		Id:          a.ID,
		Title:       a.Title,
		Description: a.Description,
		DoctorId:    a.DoctorID,
		Status:      string(a.Status),
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   a.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func mapCreateError(err error) error {
	msg := err.Error()
	switch {
	case msg == "title is required" || msg == "doctor_id is required":
		return status.Error(codes.InvalidArgument, msg)
	case strings.HasPrefix(msg, "failed to validate doctor"):
		return status.Error(codes.Unavailable, "doctor service is unreachable: "+msg)
	case msg == "doctor does not exist":
		return status.Error(codes.FailedPrecondition, "doctor does not exist")
	default:
		return status.Error(codes.Internal, msg)
	}
}

func mapUpdateError(err error) error {
	msg := err.Error()
	switch {
	case msg == "invalid status":
		return status.Error(codes.InvalidArgument, msg)
	case msg == "cannot transition from done to new":
		return status.Error(codes.InvalidArgument, msg)
	case msg == "appointment not found":
		return status.Error(codes.NotFound, msg)
	default:
		return status.Error(codes.Internal, msg)
	}
}
