package service

import (
	courierports "github.com/calledchrist/courier-dispatch/internal/app/courier/ports"
	ports "github.com/calledchrist/courier-dispatch/internal/app/delivery/ports"
	orderports "github.com/calledchrist/courier-dispatch/internal/app/order/ports"
	"github.com/calledchrist/courier-dispatch/internal/shared/uow"
)

type Dependencies struct {
	UOW        uow.UnitOfWork
	Deliveries ports.Repository
	Couriers   courierports.Repository
	Orders     orderports.Repository
}
