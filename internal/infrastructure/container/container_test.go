package container

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/calledchrist/courier-dispatch/internal/infrastructure/memory"
	"github.com/gin-gonic/gin"
)

func TestInMemoryAPIWiring(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}

	err = c.Invoke(func(router *gin.Engine, store *memory.Store) {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", "/health", nil))
		if response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}

		stores, err := store.Orders().List(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if len(stores) != 2 {
			t.Fatal("seed missing")
		}

		response = httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/dispatch", strings.NewReader(`{"order_store_id":"`+stores[0].ID().String()+`","city":"tehran"}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, req)
		if response.Code != 201 {
			t.Fatal(response.Code, response.Body.String())
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}
