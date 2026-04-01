package app

import (
	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	transporthttp "appointment-service/internal/transport/http"
	"appointment-service/internal/usecase"
	"log"

	"github.com/gin-gonic/gin"
)

func Run(port string, doctorServiceURL string) {
	repo := repository.NewInMemoryAppointmentRepository()
	doctorClient := client.NewHTTPDoctorClient(doctorServiceURL)
	uc := usecase.NewAppointmentUseCase(repo, doctorClient)
	handler := transporthttp.NewAppointmentHandler(uc)

	router := gin.Default()
	handler.RegisterRoutes(router)

	log.Printf("Appointment Service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
