package routes

import (
	"net/http"

	"github.com/seanalden-great/gycora-api/controllers"

)

// RegisterRoutes berfungsi seperti api.php di Laravel
func RegisterRoutes(mux *http.ServeMux) {
	// --- ROUTE CATEGORIES ---
	mux.HandleFunc("GET /api/categories", controllers.GetCategories)
	mux.HandleFunc("POST /api/categories", controllers.CreateCategory)
	mux.HandleFunc("PUT /api/categories/{id}", controllers.UpdateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", controllers.DeleteCategory)

	// --- ROUTE PRODUCTS ---
	mux.HandleFunc("GET /api/products", controllers.GetProducts)
	mux.HandleFunc("GET /api/products/{id}", controllers.GetProductByID)
	mux.HandleFunc("POST /api/products", controllers.CreateProduct)
	mux.HandleFunc("PUT /api/products/{id}", controllers.UpdateProduct)
	mux.HandleFunc("DELETE /api/products/{id}", controllers.DeleteProduct)

	// --- ROUTE AUTH ---
	mux.HandleFunc("POST /api/register", controllers.Register)
	mux.HandleFunc("POST /api/login", controllers.Login)
	mux.HandleFunc("POST /api/admin/login", controllers.AdminLogin)
	mux.HandleFunc("GET /api/admin/users", controllers.GetAllUsers)
	mux.HandleFunc("PUT /api/profile", controllers.UpdateProfile)

	// --- ROUTE CARTS ---
	mux.HandleFunc("GET /api/carts", controllers.GetCarts)
	mux.HandleFunc("POST /api/carts", controllers.AddToCart)
	mux.HandleFunc("PUT /api/carts/{id}", controllers.UpdateCart)
	mux.HandleFunc("DELETE /api/carts/{id}", controllers.DeleteCart)

	// --- ROUTE ADDRESSES ---
	mux.HandleFunc("GET /api/addresses", controllers.GetAddresses)
	mux.HandleFunc("POST /api/addresses", controllers.CreateAddress)
	mux.HandleFunc("PUT /api/addresses/{id}", controllers.UpdateAddress)
	mux.HandleFunc("DELETE /api/addresses/{id}", controllers.DeleteAddress)

	// --- ROUTE CONTACTS ---
	mux.HandleFunc("POST /api/contact", controllers.SubmitContact)
}
