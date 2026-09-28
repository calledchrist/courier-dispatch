package dispatch_service

import (
	"context"
	"errors"
	"strings"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
)

type ManualAssignService struct {
	workflow *Service
}

func NewManualAssignService(deps Dependencies) *ManualAssignService {
	return &ManualAssignService{workflow: New(deps)}
}

var _ ports.ManualAssignmentService = (*ManualAssignService)(nil)

// Assign evaluates only the requested courier and never substitutes another one.
func (s *ManualAssignService) Assign(ctx context.Context, req ports.ManualAssignRequest) (ports.DispatchResult, error) {
	if strings.TrimSpace(req.OrderStoreID.String()) == "" || strings.TrimSpace(req.CourierID.String()) == "" {
		return ports.DispatchResult{}, errors.New("order store ID and courier ID are required")
	}

	parameters := domain.DefaultParameters()
	if req.Parameters != nil {
		parameters = *req.Parameters
	}

	policy, err := domain.NewPolicy(parameters)
	if err != nil {
		return ports.DispatchResult{}, err
	}

	workflow := s.workflow
	var result ports.DispatchResult
	err = workflow.deps.UOW.Do(ctx, func(tx context.Context) error {
		store, err := workflow.assignableStore(tx, req.OrderStoreID)
		if err != nil {
			return err
		}

		courier, err := workflow.deps.Couriers.Get(tx, req.CourierID)
		if err != nil {
			return err
		}

		now := workflow.deps.Clock.Now()
		score, eligible, err := workflow.evaluateCourier(tx, store, courier, policy, now)
		if err != nil {
			return err
		}

		if !eligible {
			return domain.ErrCourierNotEligible
		}

		result, err = workflow.assign(tx, store, courier, score, parameters, now, domain.ManualAssignment)

		return err
	})
	if err != nil {
		return ports.DispatchResult{}, err
	}

	return result, nil
}
