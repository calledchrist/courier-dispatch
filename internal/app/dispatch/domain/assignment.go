package dispatch_domain

import (
	"errors"
	"math"
	"time"

	"github.com/calledchrist/courier-dispatch/internal/shared/ddd"
)

type AssignmentMethod string

const (
	AutomaticAssignment AssignmentMethod = "automatic"
	ManualAssignment    AssignmentMethod = "manual"
)

var ErrCourierNotEligible = errors.New("selected courier is not eligible for this order store")

// Assignment records the decision and policy that produced one delivery.
type AssignmentProps struct {
	CourierID    ddd.ID
	OrderStoreID ddd.ID
	DeliveryID   ddd.ID
	Method       AssignmentMethod
	Score        float64
	Route        RouteEstimate
	Parameters   Parameters
	AssignedAt   time.Time
}

type Assignment struct {
	aggregate ddd.Aggregate[AssignmentProps]
}

func NewAssignment(p AssignmentProps) (*Assignment, error) {
	if p.Method == "" {
		p.Method = AutomaticAssignment
	}

	a, err := ddd.NewAggregate(ddd.NewID(), p, ValidateAssignment)
	if err != nil {
		return nil, err
	}

	return &Assignment{aggregate: *a}, nil
}

func ValidateAssignment(p AssignmentProps) error {
	if p.Method != AutomaticAssignment && p.Method != ManualAssignment {
		return errors.New("invalid assignment method")
	}

	if p.CourierID.Empty() || p.OrderStoreID.Empty() || p.DeliveryID.Empty() || p.AssignedAt.IsZero() {
		return errors.New("assignment references and timestamp are required")
	}

	if math.IsNaN(p.Score) || math.IsInf(p.Score, 0) || p.Score < 0 || p.Score > 100 {
		return errors.New("invalid assignment score")
	}

	if err := p.Parameters.Validate(); err != nil {
		return err
	}

	return p.Route.Validate()
}

func (a *Assignment) ID() ddd.ID {
	return a.aggregate.ID()
}

func (a *Assignment) Props() AssignmentProps {
	return a.aggregate.Props()
}

func (a *Assignment) Clone() *Assignment {
	c := *a

	return &c
}
