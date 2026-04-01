package http

import (
	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppointmentHandler struct {
	useCase *usecase.AppointmentUseCase
}

func NewAppointmentHandler(useCase *usecase.AppointmentUseCase) *AppointmentHandler {
	return &AppointmentHandler{useCase: useCase}
}

type CreateAppointmentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DoctorID    string `json:"doctor_id"`
}

type UpdateStatusRequest struct {
	Status model.Status `json:"status"`
}

func (h *AppointmentHandler) CreateAppointment(c *gin.Context) {
	var req CreateAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appointment, err := h.useCase.CreateAppointment(req.Title, req.Description, req.DoctorID)
	if err != nil {
		log.Printf("Failed to create appointment: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, appointment)
}

func (h *AppointmentHandler) GetAppointment(c *gin.Context) {
	id := c.Param("id")

	appointment, err := h.useCase.GetAppointmentByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, appointment)
}

func (h *AppointmentHandler) GetAllAppointments(c *gin.Context) {
	appointments, err := h.useCase.GetAllAppointments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, appointments)
}

func (h *AppointmentHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	appointment, err := h.useCase.UpdateAppointmentStatus(id, req.Status)
	if err != nil {
		log.Printf("Failed to update appointment status: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, appointment)
}

func (h *AppointmentHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/appointments", h.CreateAppointment)
	router.GET("/appointments/:id", h.GetAppointment)
	router.GET("/appointments", h.GetAllAppointments)
	router.PATCH("/appointments/:id/status", h.UpdateStatus)
}
