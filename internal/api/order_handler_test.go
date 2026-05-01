package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"simple-orderbook/internal/core/domain"
)

type MockOrderService struct {
	PlaceOrderFunc   func(ctx context.Context, order *domain.Order) error
	CancelOrderFunc  func(ctx context.Context, id uuid.UUID) error
	GetOrderBookFunc func(ctx context.Context) ([]byte, error)
}

func (m *MockOrderService) PlaceOrder(ctx context.Context, order *domain.Order) error {
	return m.PlaceOrderFunc(ctx, order)
}

func (m *MockOrderService) CancelOrder(ctx context.Context, id uuid.UUID) error { return nil }
func (m *MockOrderService) GetOrderBook(ctx context.Context) ([]byte, error)    { return nil, nil }
func (m *MockOrderService) RebuildOrderBook(ctx context.Context) error          { return nil }

type MockMetrics struct{}

func (m *MockMetrics) RecordOrderPlacement(duration time.Duration, status string) { return }
func (m *MockMetrics) RecordMatchingLatency(duration time.Duration)               { return }
func (m *MockMetrics) RecordEndToEndLatency(duration time.Duration)               { return }
func (m *MockMetrics) RecordTrade(quantity int64)                                 { return }

type MockGenerator struct {
	FixedID uuid.UUID
}

func (m *MockGenerator) Next() uuid.UUID { return m.FixedID }

func TestCreateOrder(t *testing.T) {
	testUserID := uuid.New()

	mockGen := &MockGenerator{FixedID: uuid.New()}

	cases := []struct {
		name           string
		input          string
		withAuth       bool
		mockServiceErr error
		expectedCode   int
		expectedType   string
	}{
		{
			name:         "Unathorized_MissingContextID",
			input:        `{"price": 100, "quantity": 10, "side": "buy"}`,
			withAuth:     false,
			expectedCode: http.StatusUnauthorized,
			expectedType: "unauthorized",
		},
		{
			name:         "InvalidJSON",
			input:        `{"price": "high", "quantity": 10}`,
			withAuth:     true,
			expectedCode: http.StatusBadRequest,
			expectedType: "invalid-json",
		},
		{
			name:         "NegativeQuantity",
			input:        `{"price": 100, "quantity": -5, "side": "buy"}`,
			withAuth:     true,
			expectedCode: http.StatusBadRequest,
			expectedType: "validation-error",
		},
		{
			name:           "ServiceFailure",
			input:          `{"price": 100, "quantity": 10, "side": "buy"}`,
			withAuth:       true,
			mockServiceErr: errors.New("db down"),
			expectedCode:   http.StatusInternalServerError,
			expectedType:   "internal-error",
		},
		{
			name:         "ValidOrder",
			input:        `{"price": 100, "quantity": 10, "side": "buy"}`,
			withAuth:     true,
			expectedCode: http.StatusCreated,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockSvc := &MockOrderService{
				PlaceOrderFunc: func(ctx context.Context, order *domain.Order) error {
					return c.mockServiceErr
				},
				CancelOrderFunc: func(ctx context.Context, id uuid.UUID) error {
					return c.mockServiceErr
				},
				GetOrderBookFunc: func(ctx context.Context) ([]byte, error) {
					return nil, c.mockServiceErr
				},
			}

			mockMetrics := &MockMetrics{}

			req := httptest.NewRequest("POST", "/order", strings.NewReader(c.input))
			req.Header.Set("Content-Type", "application/json")

			if c.withAuth {
				ctx := context.WithValue(req.Context(), UserIDKey, testUserID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler := NewOrderHandler(mockSvc, mockMetrics, mockGen)

			handler.CreateOrder(rr, req)

			if rr.Code != c.expectedCode {
				t.Errorf("wrong status code: got %v want %v", rr.Code, c.expectedCode)
			}

			if rr.Code >= 400 {
				var apiErr APIError
				if err := json.Unmarshal(rr.Body.Bytes(), &apiErr); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}
				if apiErr.Type != c.expectedType {
					t.Errorf("wrong error type: got %q want %q", apiErr.Type, c.expectedType)
				}
			}

			if rr.Code == http.StatusCreated {
				var createdOrder domain.Order
				if err := json.Unmarshal(rr.Body.Bytes(), &createdOrder); err != nil {
					t.Fatalf("failed to decode success reposponse: %v", err)
				}
				if createdOrder.UserID != testUserID {
					t.Errorf("order has wrong userID: got %v, want %v", createdOrder.UserID, testUserID)
				}
			}
		})
	}
}
