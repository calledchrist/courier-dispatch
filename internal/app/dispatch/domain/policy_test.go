package dispatch_domain

import (
	"math"
	"testing"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

func TestEligibilityAndConstraints(t *testing.T) {
	now := time.Now()
	location, _ := geo.NewLocation(geo.LocationProps{})
	policy, err := NewPolicy(DefaultParameters())
	if err != nil {
		t.Fatal(err)
	}

	base := Candidate{
		CourierID:         "courier",
		City:              "tehran",
		Online:            true,
		HasPOSDevice:      true,
		HasCar:            true,
		Location:          location,
		LocationUpdatedAt: now,
		ActiveTripLimit:   2,
	}
	requirements := Requirements{
		City:        "tehran",
		RequiresPOS: true,
		RequiresCar: true,
	}
	route := RouteEstimate{
		PickupKM:     1,
		TotalRouteKM: 4,
		AdditionalKM: 3,
	}
	cases := []struct {
		name   string
		change func(*Candidate, *RouteEstimate)
	}{
		{"city", func(c *Candidate, r *RouteEstimate) { c.City = "other" }},

		{"offline", func(c *Candidate, r *RouteEstimate) { c.Online = false }},

		{"missing location", func(c *Candidate, r *RouteEstimate) { c.Location = geo.Location{} }},

		{"stale location", func(c *Candidate, r *RouteEstimate) { c.LocationUpdatedAt = now.Add(-6 * time.Minute) }},

		{"future location", func(c *Candidate, r *RouteEstimate) { c.LocationUpdatedAt = now.Add(time.Second) }},

		{"POS", func(c *Candidate, r *RouteEstimate) { c.HasPOSDevice = false }},

		{"car", func(c *Candidate, r *RouteEstimate) { c.HasCar = false }},

		{"capacity", func(c *Candidate, r *RouteEstimate) { c.ActiveTrips = 2 }},

		{"pickup", func(c *Candidate, r *RouteEstimate) { r.PickupKM = 6 }},

		{"route", func(c *Candidate, r *RouteEstimate) { r.TotalRouteKM = 26 }},

		{"additional route", func(c *Candidate, r *RouteEstimate) {
			r.AdditionalKM = 16
			r.TotalRouteKM = 20
		}},

		{"NaN", func(c *Candidate, r *RouteEstimate) { r.PickupKM = math.NaN() }},
	}
	if _, ok := policy.Evaluate(requirements, base, route, now); !ok {
		t.Fatal("valid candidate rejected")
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, r := base, route
			tc.change(&c, &r)
			if _, ok := policy.Evaluate(requirements, c, r, now); ok {
				t.Fatal("ineligible candidate accepted")
			}
		})
	}

	boundary := RouteEstimate{
		PickupKM:     5,
		TotalRouteKM: 25,
		AdditionalKM: 15,
	}
	if _, ok := policy.Evaluate(requirements, base, boundary, now); !ok {
		t.Fatal("inclusive limit rejected")
	}
}

func TestScoringAndStableTieBreak(t *testing.T) {
	policy, _ := NewPolicy(DefaultParameters())
	now := time.Now()
	location, _ := geo.NewLocation(geo.LocationProps{})
	c := Candidate{
		CourierID:         "a",
		City:              "tehran",
		Online:            true,
		Location:          location,
		LocationUpdatedAt: now,
		ActiveTripLimit:   3,
	}
	first, _ := policy.Evaluate(Requirements{City: "tehran"}, c, RouteEstimate{
		PickupKM:     1,
		TotalRouteKM: 4,
		AdditionalKM: 4,
	}, now)
	c.CourierID = "b"
	c.ActiveTrips = 2
	second, _ := policy.Evaluate(Requirements{City: "tehran"}, c, RouteEstimate{
		PickupKM:     2,
		TotalRouteKM: 8,
		AdditionalKM: 8,
	}, now)
	ranked := Rank([]ScoredCandidate{second, first})
	if ranked[0].CourierID != "a" || first.Score <= second.Score {
		t.Fatal("bad scoring")
	}

	first.Score = second.Score
	ranked = Rank([]ScoredCandidate{second, first})
	if ranked[0].CourierID != "a" {
		t.Fatal("unstable tie")
	}

	for _, p := range []Parameters{
		{},
		{MaxPickupKM: math.Inf(1)},
	} {
		if _, err := NewPolicy(p); err == nil {
			t.Fatal("invalid policy accepted")
		}
	}
}
