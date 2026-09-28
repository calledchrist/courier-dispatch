package memory

import (
	"context"
	"time"

	courier "github.com/calledchrist/courier-dispatch/internal/app/courier/domain"
	order "github.com/calledchrist/courier-dispatch/internal/app/order/domain"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

// SeedDemo uses domain constructors and hardcoded slices. No database is opened.
// Two stores share one external order ID and can be dispatched independently.
func SeedDemo(ctx context.Context, store *Store) error {
	return store.Do(ctx, func(tx context.Context) error {
		couriers := []courier.CreateCourierProps{
			{
				Name:            "Ali",
				Mobile:          "09120000001",
				City:            "tehran",
				Status:          courier.CourierOnline,
				VehicleType:     courier.VehicleBike,
				ActiveTripLimit: 2,
			},

			{
				Name:            "Sara",
				Mobile:          "09120000002",
				City:            "tehran",
				Status:          courier.CourierOnline,
				VehicleType:     courier.VehicleCar,
				HasPOSDevice:    true,
				ActiveTripLimit: 2,
			},

			{
				Name:         "Reza",
				Mobile:       "09120000003",
				City:         "tehran",
				Status:       courier.CourierOffline,
				VehicleType:  courier.VehicleCar,
				HasPOSDevice: true,
			},
		}
		for i, props := range couriers {
			c, err := courier.NewCourier(props)
			if err != nil {
				return err
			}

			location, err := geo.NewLocation(geo.LocationProps{
				Latitude:  35.700 + float64(i)*0.002,
				Longitude: 51.400,
			})
			if err != nil {
				return err
			}

			if err := c.UpdateLocation(location, time.Now().UTC()); err != nil {
				return err
			}

			if err := store.Couriers().Save(tx, c); err != nil {
				return err
			}
		}

		customerLocation, err := order.NewLocation(order.LocationProps{
			Latitude:  35.710,
			Longitude: 51.410,
		})
		if err != nil {
			return err
		}

		vendors := []order.VendorProps{
			{
				VendorId:    101,
				Title:       "Bakery",
				AddressText: "Demo bakery address",
				City:        "tehran",
			},
			{
				VendorId:    102,
				Title:       "Market",
				AddressText: "Demo market address",
				City:        "tehran",
			},
		}
		for i, vendor := range vendors {
			vendor.Location, err = order.NewLocation(order.LocationProps{
				Latitude:  35.701 + float64(i)*0.001,
				Longitude: 51.401,
			})
			if err != nil {
				return err
			}

			props := order.CreateOrderStoreProps{
				OrderStoreId:    uint(1001 + i),
				City:            "tehran",
				RequiresPOS:     i == 1,
				PreparationTime: time.Now().UTC(),
			}
			if i == 1 {
				props.VehicleNeeded = order.VehicleCar
			}

			aggregate, err := order.NewOrderStore(props)
			if err != nil {
				return err
			}

			if err := aggregate.AddOrder(order.OrderProps{
				OrderId: "demo-order-1",
				Total:   250000,
			}); err != nil {
				return err
			}

			if err := aggregate.AddCustomer(order.CustomerProps{
				Name:        "Demo customer",
				Mobile:      "09120000009",
				AddressText: "Demo drop-off address",
				Location:    customerLocation,
			}); err != nil {
				return err
			}

			if err := aggregate.AddVendor(vendor); err != nil {
				return err
			}

			if err := aggregate.AddItems(vendor.VendorId, []order.ItemProps{{
				Title: "Demo item",
				Count: 2,
				Unit:  "piece",
			}}); err != nil {
				return err
			}

			if err := aggregate.MarkReady(time.Now().UTC()); err != nil {
				return err
			}

			if err := store.Orders().Save(tx, aggregate); err != nil {
				return err
			}
		}

		return nil
	})
}
