package httpserver

import (
	deliveryhandler "github.com/calledchrist/courier-dispatch/internal/app/delivery/handler"
	dispatchhandler "github.com/calledchrist/courier-dispatch/internal/app/dispatch/handler"
	"github.com/gin-gonic/gin"
)

func RegisterDomains(
	router *gin.Engine,
	dispatcher *dispatchhandler.DispatchHandler,
	manual *dispatchhandler.ManualAssignHandler,
	deliveries *deliveryhandler.DeliveryHandler,
) {
	router.POST("/dispatch", dispatcher.Dispatch)
	router.POST("/dispatch/manual", manual.Assign)
	router.GET("/couriers/:id/deliveries", deliveries.Active)
	router.POST("/deliveries/:id/status", deliveries.Transition)
}
