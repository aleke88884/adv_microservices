package main

import (
	"doctor-service/internal/app"
	"os"
)

func main() {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}
	app.Run(port)
}
