package main

import (
	"appointment-service/internal/app"
	"os"
)

func main() {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50052"
	}
	doctorAddr := os.Getenv("DOCTOR_SERVICE_ADDR")
	if doctorAddr == "" {
		doctorAddr = "localhost:50051"
	}
	app.Run(port, doctorAddr)
}
