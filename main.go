package main

import (
	"lugbit/projects/checkout/database"
	product "lugbit/projects/checkout/product"

	"github.com/gin-gonic/gin"
)

func main() {
	route := gin.Default()
	// connect to DB
	database.ConnectDatabase()

	// routes
	const (
		productURL  = "/product"
		purchaseURL = "/purchase"
	)

	// list products GET
	route.GET(productURL, product.ListProducts)
	// add product POST
	route.POST(productURL, product.AddProduct)
	// purchase item POST
	route.POST(purchaseURL, product.PurchaseItems)

	err := route.Run(":8080")
	if err != nil {
		panic(err)
	}
}
