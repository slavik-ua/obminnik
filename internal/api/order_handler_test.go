package api

import (
	"context"
	"strings"
	"errors"
	"encoding/json"
	"simple-orderbook/internal/core/domain"

	"testing"
	"net/http"
	"net/http/httptest"
)

type MockOrderRepository struct {
	CreateFunc func(ctx context.Context, order *domain.Order) error
}

func (m *MockOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	return m.CreateFunc(ctx, order)
}

func TestCreateOrder(t *testing.T) {
	cases := []struct{
		name          string
		input         string
		price         int64
		quantity      int64
		side          domain.OrderSide
		mockRepoErr   error
		expectedCode  int
	}{
		{
			name:        "NegativeQuantity",
			input:       `{"price": 100, "quantity": -5, "side": "BUY"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "NegativePrice",
			input:        `{"price": -100, "quantity": 10, "side": "BUY"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "WrongSide",
			input:        `{"price": 100, "quantity": 10, "side": "asd"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "DatabaseFailure",
			input:        `{"price": 100, "quantity": 10, "side": "BUY"}`,
			mockRepoErr:  errors.New("connection refused"),
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "ValidOrder",
			input:        `{"price": 100, "quantity": 10, "side": "BUY"}`,
			price:        100,
			quantity:     10,
			side:         domain.SideBuy,
			mockRepoErr:  nil,
			expectedCode: http.StatusCreated,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				CreateFunc: func(ctx context.Context, order *domain.Order) error {
					return c.mockRepoErr
				},
			}

			req := httptest.NewRequest("POST", "/order", strings.NewReader(c.input))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler := NewOrderHandler(mockRepo)
			handler.CreateOrder(rr, req)

			if rr.Code != c.expectedCode {
				t.Errorf("wrong status code: got %v want %v", rr.Code, c.expectedCode)
			}

			if rr.Code == http.StatusCreated {
				var createdOrder domain.Order

				dec := json.NewDecoder(rr.Body)
				dec.DisallowUnknownFields()

				if err := dec.Decode(&createdOrder); err != nil {
					t.Errorf("error decoding: %v", err)
				}

				if createdOrder.Price != c.price {
					t.Errorf("wrong price: got %v, want %v", createdOrder.Price, c.price)
				}

				if createdOrder.Quantity != c.quantity {
					t.Errorf("wrong quantity: got %v, want %v", createdOrder.Quantity, c.quantity)
				}

				if createdOrder.Side != c.side {
					t.Errorf("wrong side: got %v, want %v", createdOrder.Side, c.side)
				}
			}
		})
	}
}