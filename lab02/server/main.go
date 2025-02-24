package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"os"
	"path/filepath"
)

type Product struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ID          string `json:"id"`
	Image       string `json:"image,omitempty"`
}

func IsEqual(a, b Product) bool {
	return a.Name == b.Name && a.Description == b.Description
}

func BindBasic(c *gin.Context, newProduct *Product) bool {
	name, err := c.GetPostForm("name")
	if !err {
		return false
	}

	description, err := c.GetPostForm("description")
	if !err {
		return false
	}

	newProduct.Name = name
	newProduct.Description = description
	return true
}

var products []Product

func main() {

	r := gin.Default()

	r.GET("/products", getProducts)
	r.GET("/products/:id", getProductByID)
	r.POST("/products", createProduct)
	r.PUT("/products/:id", updateProduct)
	r.DELETE("/products/:id", deleteProduct)

	err := r.Run(":8080")
	if err != nil {
		os.Exit(1)
	}

}

func getProducts(c *gin.Context) {
	c.JSON(http.StatusOK, products)
}

func getProductByID(c *gin.Context) {
	id := c.Param("id")
	for _, product := range products {
		if product.ID == id {
			c.JSON(http.StatusOK, product)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "Product not found"})
}
func createProduct(c *gin.Context) {
	var newProduct Product

	if !BindBasic(c, &newProduct) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can`t get name or description"})
		return
	}

	file, _ := c.FormFile("image")
	if file != nil {
		imagePath := filepath.Join("uploads", uuid.New().String()+filepath.Ext(file.Filename))

		if err := os.MkdirAll("uploads", os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create upload directory"})
			return
		}

		if err := c.SaveUploadedFile(file, imagePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to save image"})
			return
		}

		newProduct.Image = imagePath
	}

	for _, p := range products {
		if IsEqual(newProduct, p) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Product already exists"})
			return
		}
	}

	newProduct.ID = uuid.New().String()

	products = append(products, newProduct)

	c.JSON(http.StatusCreated, newProduct)
}

func updateProduct(c *gin.Context) {
	id := c.Param("id")
	var updatedProduct Product
	BindBasic(c, &updatedProduct)

	file, _ := c.FormFile("image")
	if file != nil {
		imagePath := filepath.Join("uploads", uuid.New().String()+filepath.Ext(file.Filename))

		if err := os.MkdirAll("uploads", os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create upload directory"})
			return
		}

		if err := c.SaveUploadedFile(file, imagePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to save image"})
			return
		}

		updatedProduct.Image = imagePath
	}

	for i, product := range products {
		if product.ID == id {
			if products[i].Name != updatedProduct.Name && updatedProduct.Name != "" {
				products[i].Description = updatedProduct.Description
			}
			if products[i].Image != updatedProduct.Image && updatedProduct.Image != "" {
				products[i].Image = updatedProduct.Image
			}
			c.JSON(http.StatusOK, products[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "Product not found"})
}

func deleteProduct(c *gin.Context) {
	id := c.Param("id")
	for i, product := range products {
		if product.ID == id {
			if product.Image != "" {
				if err := os.Remove(product.Image); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
					return
				}
			}

			products = append(products[:i], products[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"message": "Product not found"})
}
