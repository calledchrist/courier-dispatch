package container

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domain "github.com/calledchrist/courier-dispatch/internal/app/dispatch/domain"
	ports "github.com/calledchrist/courier-dispatch/internal/app/dispatch/ports"
	"github.com/calledchrist/courier-dispatch/internal/infrastructure/memory"
	"github.com/gin-gonic/gin"
)

func TestManualAssignmentAPI(t *testing.T) {
	container, err := New()
	if err != nil {
		t.Fatal(err)
	}

	err = container.Invoke(func(router *gin.Engine, storage *memory.Store) {
		for _, body := range []string{`{`, `{}`, `{"order_store_id":"store"}`} {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/dispatch/manual", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected invalid body rejection, got %d: %s", response.Code, response.Body.String())
			}
		}

		ctx := context.Background()
		stores, err := storage.Orders().List(ctx)
		if err != nil {
			t.Fatal(err)
		}

		couriers, err := storage.Couriers().ListByCity(ctx, "tehran")
		if err != nil {
			t.Fatal(err)
		}

		body, err := json.Marshal(map[string]string{
			"order_store_id": stores[0].ID().String(),
			"courier_id":     couriers[0].ID().String(),
		})
		if err != nil {
			t.Fatal(err)
		}

		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/dispatch/manual", strings.NewReader(string(body)))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)

		if response.Code != http.StatusCreated {
			t.Fatalf("manual assignment failed: %d: %s", response.Code, response.Body.String())
		}

		var result ports.DispatchResult
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}

		if result.CourierID != couriers[0].ID() || result.Method != domain.ManualAssignment || result.DeliveryID.Empty() {
			t.Fatalf("unexpected result: %+v", result)
		}

		response = httptest.NewRecorder()
		request = httptest.NewRequest(http.MethodPost, "/dispatch/manual", strings.NewReader(string(body)))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)

		if response.Code != http.StatusConflict {
			t.Fatalf("duplicate request should conflict, got %d", response.Code)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}
