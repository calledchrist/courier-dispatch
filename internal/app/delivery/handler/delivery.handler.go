package handler

import (
	"net/http"
	"time"

	domain "github.com/calledchrist/courier-dispatch/internal/app/delivery/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/delivery/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/gin-gonic/gin"
)

type DeliveryHandler struct {
	service ports.DeliveryServiceInboundPort
}

func NewDeliveryHandler(service ports.DeliveryServiceInboundPort) *DeliveryHandler {
	return &DeliveryHandler{service: service}
}

func (h *DeliveryHandler) Active(c *gin.Context) {
	trips, err := h.service.ActiveDeliveries(c.Request.Context(), ddd.ID(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	result := make([]gin.H, 0, len(trips))
	for _, trip := range trips {
		p := trip.Props()
		result = append(result, gin.H{
			"id":             trip.ID(),
			"courier_id":     p.CourierID,
			"order_store_id": p.OrderStoreID,
			"status":         p.Status,
			"assigned_at":    p.AssignedAt,
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *DeliveryHandler) Transition(c *gin.Context) {
	var body struct {
		Status domain.Status `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	if err := h.service.Transition(c.Request.Context(), ddd.ID(c.Param("id")), body.Status, time.Now().UTC()); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		return
	}

	c.Status(http.StatusNoContent)
}
