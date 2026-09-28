package handler

import (
	"net/http"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/gin-gonic/gin"
)

type DispatchHandler struct {
	service ports.Service
}

type dispatchBody struct {
	OrderStoreID ddd.ID             `json:"order_store_id"`
	City         string             `json:"city"`
	Parameters   *domain.Parameters `json:"parameters"`
}

func NewDispatchHandler(service ports.Service) *DispatchHandler {
	return &DispatchHandler{service: service}
}

func (h *DispatchHandler) Dispatch(c *gin.Context) {
	var body dispatchBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	parameters := domain.DefaultParameters()
	if body.Parameters != nil {
		parameters = *body.Parameters
	}

	result, err := h.service.Dispatch(c.Request.Context(), ports.DispatchRequest{
		OrderStoreID: body.OrderStoreID,
		City:         body.City,
		Parameters:   parameters,
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusCreated, result)
}
