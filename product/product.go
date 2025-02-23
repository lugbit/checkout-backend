package product

import (
	"encoding/json"
	"fmt"
	"lugbit/projects/checkout/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Product struct {
	SKU   string  `json:"sku"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Qty   float64 `json:"qty"`
}

type PurchaseItem struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

type PurchaseRequest struct {
	UserID string         `json:"user_id"`
	Items  []PurchaseItem `json:"items"`
}

type PurchaseResponse struct {
	UserID         string         `json:"user_id"`
	ItemsPurchased []PurchaseItem `json:"items_purchased"`
	TotalPrice     float64        `json:"total_price"`
}

// ListProducts lists all available products in the database.
func ListProducts(ctx *gin.Context) {
	query := "SELECT sku, name, price, qty FROM product"
	rows, err := database.Db.Query(query)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, "error with fetching products")
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.SKU, &p.Name, &p.Price, &p.Qty); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, "error scanning into struct: "+err.Error())
			return
		}
		products = append(products, p)
	}

	if err = rows.Err(); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, "error processing rows")
		return
	}

	ctx.JSON(http.StatusOK, products)
}

// AddProduct adds a new product to the database. If the SKU already exists, insert will fail.
func AddProduct(ctx *gin.Context) {
	body := Product{}
	data, err := ctx.GetRawData()
	if err != nil {
		ctx.AbortWithStatusJSON(400, "product cannot be empty")
		return
	}

	err = json.Unmarshal(data, &body)
	if err != nil {
		ctx.AbortWithStatusJSON(400, "unable to marshal JSON")
		return
	}

	query := "INSERT INTO product (sku, name, price, qty) VALUES ($1, $2, $3, $4)"

	stmt, err := database.Db.Prepare(query)
	if err != nil {
		ctx.AbortWithStatusJSON(400, "error with preparing SQL")
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(body.SKU, body.Name, body.Price, body.Qty)
	if err != nil {
		fmt.Println(err)
		ctx.AbortWithStatusJSON(400, "unable to add new product")
	} else {
		ctx.JSON(http.StatusOK, "product successfully added")
	}
}

// PurchaseItems "purchases" one or more items from the inventory. If inventory stock of the item
// is less than being purchased, the purchase will fail.
//
// Otherwise, if the purchase is successful, the inventory will be updated and the total items and
// total cost will be sent back to the API caller.
func PurchaseItems(ctx *gin.Context) {
	var req PurchaseRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	// user must be logged in to purchase
	if req.UserID == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "user id required"})
		return
	}

	if len(req.Items) == 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "no items provided"})
		return
	}

	// start db transaction so that we can roll back if any of the queries fail.
	tx, err := database.Db.Begin()
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not start transaction"})
		return
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	totalPrice := 0.0

	// loop over each purchase items in the request
	for _, item := range req.Items {
		var price float64
		var availableQty int

		// use FOR UPDATE to lock in the row to avoid other processes from updating this row
		// during this transaction which could introduce race conditions.
		row := tx.QueryRow("SELECT price, qty FROM product WHERE sku = $1 FOR UPDATE", item.SKU)
		if err := row.Scan(&price, &availableQty); err != nil {
			_ = tx.Rollback()
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("product not found or error scanning for sku: %s", item.SKU),
			})
			return
		}

		// check inventory quantity for the item(s) being purchased.
		if availableQty < item.Qty {
			_ = tx.Rollback()
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("insufficient quantity for sku: %s", item.SKU),
			})
			return
		}

		// update the inventory quantity for purchased items.
		_, err = tx.Exec("UPDATE product SET qty = qty - $1 WHERE sku = $2", item.Qty, item.SKU)
		if err != nil {
			_ = tx.Rollback()
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("failed to update quantity for sku: %s", item.SKU),
			})
			return
		}

		totalPrice += price * float64(item.Qty)
	}

	if err := tx.Commit(); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "transaction commit failed"})
		return
	}

	response := PurchaseResponse{
		UserID:         req.UserID,
		ItemsPurchased: req.Items,
		TotalPrice:     totalPrice,
	}

	ctx.JSON(http.StatusOK, response)
}
