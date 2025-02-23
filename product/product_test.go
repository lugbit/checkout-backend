package product

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lugbit/projects/checkout/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestPurchaseItems(t *testing.T) {
	var userID = "999"
	tests := []struct {
		name             string
		requestBody      PurchaseRequest
		mockSetup        func(mock sqlmock.Sqlmock)
		expectedStatus   int
		expectedResponse string
	}{
		{
			name: "user ID is empty (user not logged in)",
			requestBody: PurchaseRequest{
				UserID: "",
				Items: []PurchaseItem{
					{SKU: "120P90", Qty: 2},
					{SKU: "43N23P", Qty: 1},
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: `{"error":"user id required"}`,
		},
		{
			name: "successful purchase of multiple items",
			requestBody: PurchaseRequest{
				UserID: userID,
				Items: []PurchaseItem{
					{SKU: "120P90", Qty: 2},
					{SKU: "43N23P", Qty: 1},
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				// For "120P90": available 5, price 10.0
				rows1 := sqlmock.NewRows([]string{"price", "qty"}).AddRow(10.0, 5)
				mock.ExpectQuery("SELECT price, qty FROM product WHERE sku = \\$1 FOR UPDATE").
					WithArgs("120P90").
					WillReturnRows(rows1)
				mock.ExpectExec("UPDATE product SET qty = qty - \\$1 WHERE sku = \\$2").
					WithArgs(2, "120P90").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// For "43N23P": available 2, price 20.0
				rows2 := sqlmock.NewRows([]string{"price", "qty"}).AddRow(20.0, 2)
				mock.ExpectQuery("SELECT price, qty FROM product WHERE sku = \\$1 FOR UPDATE").
					WithArgs("43N23P").
					WillReturnRows(rows2)
				mock.ExpectExec("UPDATE product SET qty = qty - \\$1 WHERE sku = \\$2").
					WithArgs(1, "43N23P").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
			// Total price: (10.0 * 2) + (20.0 * 1) = 40.0.
			expectedResponse: `{"user_id":"999","items_purchased":[{"sku":"120P90","qty":2},{"sku":"43N23P","qty":1}],"total_price":40}`,
		},
		{
			name: "insufficient quantity for item",
			requestBody: PurchaseRequest{
				UserID: userID,
				Items: []PurchaseItem{
					{SKU: "120P90", Qty: 3},
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				// "120P90" available qty 2, but request is 3
				rows := sqlmock.NewRows([]string{"price", "qty"}).AddRow(10.0, 2)
				mock.ExpectQuery("SELECT price, qty FROM product WHERE sku = \\$1 FOR UPDATE").
					WithArgs("120P90").
					WillReturnRows(rows)
				mock.ExpectRollback()
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: `{"error":"insufficient quantity for sku: 120P90"}`,
		},
		{
			name: "product not found",
			requestBody: PurchaseRequest{
				UserID: userID,
				Items: []PurchaseItem{
					{SKU: "UNKNOWN", Qty: 1},
				},
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT price, qty FROM product WHERE sku = \\$1 FOR UPDATE").
					WithArgs("UNKNOWN").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: `{"error":"product not found or error scanning for sku: UNKNOWN"}`,
		},
		{
			name:        "invalid JSON body",
			requestBody: PurchaseRequest{
				// We'll override the JSON marshalling to send invalid JSON in this case.
			},
			mockSetup:        func(mock sqlmock.Sqlmock) {},
			expectedStatus:   http.StatusBadRequest,
			expectedResponse: `{"error":"invalid JSON body"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			// initialize mock db
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock database: %s", err)
			}
			defer db.Close()
			database.Db = db

			tc.mockSetup(mock)

			gin.SetMode(gin.TestMode)
			router := gin.Default()
			router.POST("/purchase", PurchaseItems)

			var reqBody []byte
			// this is to simulate invalid json body test case
			if tc.name == "invalid JSON body" {
				reqBody = []byte("invalid-json")
			} else {
				reqBody, err = json.Marshal(tc.requestBody)
				if err != nil {
					t.Fatalf("error marshalling request body: %s", err)
				}
			}

			req, err := http.NewRequest(http.MethodPost, "/purchase", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("failed to create HTTP request: %s", err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rr.Code)
			}

			body := rr.Body.String()
			if body != tc.expectedResponse {
				t.Errorf("expected response %s, got %s", tc.expectedResponse, body)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
