package dispatch_domain

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
	"github.com/calledchrist/courier-dispatch/internal/shared/geo"
)

var ErrNoEligibleCourier = errors.New("no eligible courier")

type Requirements struct {
	City        string
	RequiresPOS bool
	RequiresCar bool
}

// Candidate is a dispatch-owned snapshot, not a courier aggregate.
type Candidate struct {
	CourierID         ddd.ID
	City              string
	Online            bool
	HasPOSDevice      bool
	HasCar            bool
	Location          geo.Location
	LocationUpdatedAt time.Time
	ActiveTrips       int
	ActiveTripLimit   int
}

type RouteEstimate struct {
	PickupKM     float64
	TotalRouteKM float64
	AdditionalKM float64
}

func (r RouteEstimate) Validate() error {
	for _, v := range []float64{r.PickupKM, r.TotalRouteKM, r.AdditionalKM} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return errors.New("invalid route estimate")
		}
	}

	if r.AdditionalKM > r.TotalRouteKM {
		return errors.New("additional distance exceeds route distance")
	}

	return nil
}

type Parameters struct {
	MaxPickupKM     float64
	MaxRouteKM      float64
	MaxAdditionalKM float64
	MaxLocationAge  time.Duration
	PickupWeight    float64
	RouteWeight     float64
	LoadWeight      float64
}

func DefaultParameters() Parameters {
	return Parameters{
		MaxPickupKM:     5,
		MaxRouteKM:      25,
		MaxAdditionalKM: 15,
		MaxLocationAge:  5 * time.Minute,
		PickupWeight:    0.5,
		RouteWeight:     0.3,
		LoadWeight:      0.2,
	}
}

func (p Parameters) Validate() error {
	for _, v := range []float64{p.MaxPickupKM, p.MaxRouteKM, p.MaxAdditionalKM} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			return errors.New("distance limits must be finite and positive")
		}
	}

	sum := 0.0
	for _, v := range []float64{p.PickupWeight, p.RouteWeight, p.LoadWeight} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return errors.New("weights must be finite and nonnegative")
		}

		sum += v
	}

	if sum <= 0 || math.IsInf(sum, 0) || p.MaxLocationAge <= 0 {
		return errors.New("positive weights and location age limit are required")
	}

	return nil
}

type Policy struct {
	parameters Parameters
}

func NewPolicy(p Parameters) (Policy, error) {
	if err := p.Validate(); err != nil {
		return Policy{}, err
	}

	return Policy{parameters: p}, nil
}

func (p Policy) Eligible(requirements Requirements, candidate Candidate, now time.Time) bool {
	if candidate.CourierID.Empty() || strings.TrimSpace(requirements.City) == "" {
		return false
	}

	if candidate.City != requirements.City || !candidate.Online {
		return false
	}

	if candidate.Location.Validate() != nil || candidate.LocationUpdatedAt.IsZero() {
		return false
	}

	if candidate.LocationUpdatedAt.After(now) || now.Sub(candidate.LocationUpdatedAt) > p.parameters.MaxLocationAge {
		return false
	}

	if requirements.RequiresPOS && !candidate.HasPOSDevice {
		return false
	}

	if requirements.RequiresCar && !candidate.HasCar {
		return false
	}

	return candidate.ActiveTrips >= 0 &&
		candidate.ActiveTripLimit > 0 &&
		candidate.ActiveTrips < candidate.ActiveTripLimit
}

type ScoredCandidate struct {
	CourierID ddd.ID
	Score     float64
	Route     RouteEstimate
}

func (p Policy) Evaluate(r Requirements, c Candidate, route RouteEstimate, now time.Time) (ScoredCandidate, bool) {
	params := p.parameters
	if params.Validate() != nil || !p.Eligible(r, c, now) || route.Validate() != nil {
		return ScoredCandidate{}, false
	}

	if route.PickupKM > params.MaxPickupKM ||
		route.TotalRouteKM > params.MaxRouteKM ||
		route.AdditionalKM > params.MaxAdditionalKM {
		return ScoredCandidate{}, false
	}

	pickupPenalty := params.PickupWeight * (route.PickupKM / params.MaxPickupKM)
	routePenalty := params.RouteWeight * (route.TotalRouteKM / params.MaxRouteKM)
	loadPenalty := params.LoadWeight * (float64(c.ActiveTrips) / float64(c.ActiveTripLimit))
	totalWeight := params.PickupWeight + params.RouteWeight + params.LoadWeight
	penalty := (pickupPenalty + routePenalty + loadPenalty) / totalWeight

	return ScoredCandidate{
		CourierID: c.CourierID,
		Score:     100 * (1 - penalty),
		Route:     route,
	}, true
}

func Rank(candidates []ScoredCandidate) []ScoredCandidate {
	ranked := append([]ScoredCandidate(nil), candidates...)
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].CourierID < ranked[j].CourierID
		}

		return ranked[i].Score > ranked[j].Score
	})

	return ranked
}
