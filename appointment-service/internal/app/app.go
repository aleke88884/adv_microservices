package app

import (
	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	transportgrpc "appointment-service/internal/transport/grpc"
	"appointment-service/internal/usecase"
	pb "appointment-service/proto"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Run(port string, doctorServiceAddr string) {
	repo := repository.NewInMemoryAppointmentRepository()

	doctorClient, err := client.NewGRPCDoctorClient(doctorServiceAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Doctor Service at %s: %v", doctorServiceAddr, err)
	}

	uc := usecase.NewAppointmentUseCase(repo, doctorClient)
	server := transportgrpc.NewAppointmentServer(uc)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterAppointmentServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	log.Printf("Appointment Service gRPC starting on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
