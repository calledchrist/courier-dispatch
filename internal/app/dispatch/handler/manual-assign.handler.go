package handler

import (
	"net/http"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/gin-gonic/gin"
)

type ManualAssignHandler struct {
	service ports.ManualAssignmentService
}

type manualAssignBody struct {
	OrderStoreID ddd.ID             `json:"order_store_id" binding:"required"`
	CourierID    ddd.ID             `json:"courier_id" binding:"required"`
	Parameters   *domain.Parameters `json:"parameters"`
}

func NewManualAssignHandler(service ports.ManualAssignmentService) *ManualAssignHandler {
	return &ManualAssignHandler{service: service}
}

func (h *ManualAssignHandler) Assign(c *gin.Context) {
	var body manualAssignBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	result, err := h.service.Assign(c.Request.Context(), ports.ManualAssignRequest{
		OrderStoreID: body.OrderStoreID,
		CourierID:    body.CourierID,
		Parameters:   body.Parameters,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusCreated, result)
}
