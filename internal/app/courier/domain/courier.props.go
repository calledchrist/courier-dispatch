package courier_domain

import (
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

type CourierStatus string

const (
	CourierOffline   CourierStatus = "offline"
	CourierOnline    CourierStatus = "online"
	CourierSuspended CourierStatus = "suspended"
)

type VehicleType string

const (
	VehicleBike VehicleType = "bike"
	VehicleCar  VehicleType = "car"
)

const DefaultActiveTripLimit = 3

type Location = geo.Location

type LocationProps = geo.LocationProps

func NewLocation(p LocationProps) (Location, error) {
	return geo.NewLocation(p)
}

type CourierProps struct {
	Name              string
	Mobile            string
	Avatar            string
	City              string
	HasPOSDevice      bool
	Status            CourierStatus
	VehicleType       VehicleType
	Location          Location
	LocationUpdatedAt time.Time
	ActiveTripLimit   int
	ActiveDeliveryIDs []ddd.ID
}

type CreateCourierProps struct {
	Name            string
	Mobile          string
	Avatar          string
	City            string
	HasPOSDevice    bool
	Status          CourierStatus
	VehicleType     VehicleType
	ActiveTripLimit int
}
