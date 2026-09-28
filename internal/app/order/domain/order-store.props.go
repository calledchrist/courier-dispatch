package order_domain

import "time"

type VehicleType string

const (
	VehicleBike VehicleType = "bike"
	VehicleCar  VehicleType = "car"
)

type CreateOrderStoreProps struct {
	OrderStoreId    uint
	City            string
	VehicleNeeded   VehicleType
	IsCash          bool
	RequiresPOS     bool
	PreparationTime time.Time
}

type OrderStoreProps struct {
	OrderStoreId    uint
	City            string
	VehicleNeeded   VehicleType
	IsCash          bool
	RequiresPOS     bool
	PreparationTime time.Time
	Order           Order
	Vendor          Vendor
	Status          Status
	History         []OrderStatus
}
