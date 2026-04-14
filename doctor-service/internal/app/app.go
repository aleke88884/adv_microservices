package app

import (
	"doctor-service/internal/repository"
	transportgrpc "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"
	pb "doctor-service/proto"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Run(port string) {
	repo := repository.NewInMemoryDoctorRepository()
	uc := usecase.NewDoctorUseCase(repo)
	server := transportgrpc.NewDoctorServer(uc)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDoctorServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	log.Printf("Doctor Service gRPC starting on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
