package container

import (
	"context"
	"log/slog"
	"os"

	deliveryhandler "github.com/calledchrist/courier-dispatch/internal/app/delivery/handler"
	deliveryservice "github.com/calledchrist/courier-dispatch/internal/app/delivery/service"
	dispatchhandler "github.com/calledchrist/courier-dispatch/internal/app/dispatch/handler"
	dispatchservice "github.com/calledchrist/courier-dispatch/internal/app/dispatch/service"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/config"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/httpserver"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/memory"
	"github.com/gin-gonic/gin"
	"go.uber.org/dig"
)

type Container struct {
	digContainer *dig.Container
}

func New() (*Container, error) {
	container := dig.New()
	providers := []any{
		config.Load,
		newLogger,
		newMemoryStore,
		newDispatchDependencies,
		newDispatchService,
		dispatchservice.NewManualAssignService,
		newDeliveryService,
		newRouter,
	}

	for _, provider := range providers {
		if err := container.Provide(provider); err != nil {
			return nil, err
		}
	}

	return &Container{digContainer: container}, nil
}

func (c *Container) Invoke(function any) error {
	return c.digContainer.Invoke(function)
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func newMemoryStore() (*memory.Store, error) {
	store := memory.New()
	if err := memory.SeedDemo(context.Background(), store); err != nil {
		return nil, err
	}

	return store, nil
}

func newDispatchDependencies(store *memory.Store) dispatchservice.Dependencies {
	return dispatchservice.Dependencies{
		UOW:         store,
		Orders:      store.Orders(),
		Couriers:    store.Couriers(),
		Deliveries:  store.Deliveries(),
		Assignments: store.Assignments(),
		Router:      memory.StraightLineRouter{},
		Clock:       memory.Clock{},
	}
}

func newDispatchService(deps dispatchservice.Dependencies) *dispatchservice.Service {
	return dispatchservice.New(deps)
}

func newDeliveryService(store *memory.Store) *deliveryservice.DeliveryService {
	return deliveryservice.NewDeliveryService(deliveryservice.Dependencies{
		UOW:        store,
		Orders:     store.Orders(),
		Couriers:   store.Couriers(),
		Deliveries: store.Deliveries(),
	})
}

func newRouter(
	cfg *config.Config,
	dispatcher *dispatchservice.Service,
	manual *dispatchservice.ManualAssignService,
	deliveries *deliveryservice.DeliveryService,
) *gin.Engine {
	router := httpserver.New(cfg)
	dispatchHandler := dispatchhandler.NewDispatchHandler(dispatcher)
	manualHandler := dispatchhandler.NewManualAssignHandler(manual)
	deliveryHandler := deliveryhandler.NewDeliveryHandler(deliveries)
	httpserver.RegisterDomains(router, dispatchHandler, manualHandler, deliveryHandler)

	return router
}
