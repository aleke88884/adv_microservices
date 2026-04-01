package http

import (
	"doctor-service/internal/usecase"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DoctorHandler struct {
	useCase *usecase.DoctorUseCase
}

func NewDoctorHandler(useCase *usecase.DoctorUseCase) *DoctorHandler {
	return &DoctorHandler{useCase: useCase}
}

type CreateDoctorRequest struct {
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

func (h *DoctorHandler) CreateDoctor(c *gin.Context) {
	var req CreateDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doctor, err := h.useCase.CreateDoctor(req.FullName, req.Specialization, req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doctor)
}

func (h *DoctorHandler) GetDoctor(c *gin.Context) {
	id := c.Param("id")

	doctor, err := h.useCase.GetDoctorByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doctor)
}

func (h *DoctorHandler) GetAllDoctors(c *gin.Context) {
	doctors, err := h.useCase.GetAllDoctors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doctors)
}

func (h *DoctorHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/doctors", h.CreateDoctor)
	router.GET("/doctors/:id", h.GetDoctor)
	router.GET("/doctors", h.GetAllDoctors)
}
