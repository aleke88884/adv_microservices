package app

import (
	"doctor-service/internal/repository"
	transporthttp "doctor-service/internal/transport/http"
	"doctor-service/internal/usecase"
	"log"

	"github.com/gin-gonic/gin"
)

func Run(port string) {
	repo := repository.NewInMemoryDoctorRepository()
	uc := usecase.NewDoctorUseCase(repo)
	handler := transporthttp.NewDoctorHandler(uc)

	router := gin.Default()
	handler.RegisterRoutes(router)

	log.Printf("Doctor Service starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
