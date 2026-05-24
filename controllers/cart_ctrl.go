package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/seanalden-great/gycora-api/config"
)

// Struktur Data Keranjang (Mirip Eloquent Relationship with('product'))
type Cart struct {
	ID          int      `json:"id"`
	UserID      int      `json:"user_id"`
	ProductID   int      `json:"product_id"`
	Quantity    int      `json:"quantity"`
	GrossAmount float64  `json:"gross_amount"`
	Product     *Product `json:"product,omitempty"` // Relasi ke struct Product yang ada di product_ctrl.go
}

type CartRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// Helper: Ekstrak ID User dari Header Authorization (Pengganti $request->user()->id)
func getUserIDFromToken(r *http.Request) (int, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0, fmt.Errorf("missing authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return jwtSecretKey, nil // Menggunakan jwtSecretKey dari auth_ctrl.go
	})

	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid claims")
	}

	userID := int(claims["user_id"].(float64))
	return userID, nil
}

// =========================================================================
// 1. GET: Ambil Semua Isi Keranjang (index)
// =========================================================================
func GetCarts(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Query JOIN antara carts dan products (Meniru with('product') di Laravel)
	query := `
		SELECT c.id, c.user_id, c.product_id, c.quantity, c.gross_amount, 
		       p.sku, p.name, p.price, p.stock, p.image_url 
		FROM carts c
		JOIN products p ON c.product_id = p.id
		WHERE c.user_id = ?
		ORDER BY c.id DESC
	`
	rows, err := config.DB.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var carts []Cart
	for rows.Next() {
		var c Cart
		var p Product
		var imgUrl sql.NullString

		err := rows.Scan(&c.ID, &c.UserID, &c.ProductID, &c.Quantity, &c.GrossAmount,
			&p.SKU, &p.Name, &p.Price, &p.Stock, &imgUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		p.ID = c.ProductID
		p.ImageURL = imgUrl.String
		c.Product = &p
		carts = append(carts, c)
	}

	// Jika kosong, kembalikan array kosong bukan null
	if len(carts) == 0 {
		carts = []Cart{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(carts)
}

// =========================================================================
// 2. POST: Tambah ke Keranjang (store)
// =========================================================================
func AddToCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Quantity < 1 {
		http.Error(w, "Quantity must be at least 1", http.StatusUnprocessableEntity)
		return
	}

	// Ambil data produk (Stok dan Harga)
	var stock int
	var price float64
	err = config.DB.QueryRow("SELECT stock, price FROM products WHERE id = ?", req.ProductID).Scan(&stock, &price)
	if err == sql.ErrNoRows {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Cek apakah produk sudah ada di keranjang user ini
	var cartID, existingQty int
	err = config.DB.QueryRow("SELECT id, quantity FROM carts WHERE user_id = ? AND product_id = ?", userID, req.ProductID).Scan(&cartID, &existingQty)

	newQuantity := req.Quantity
	isUpdate := false

	if err != sql.ErrNoRows {
		// Produk sudah ada, tambahkan quantity lamanya
		newQuantity += existingQty
		isUpdate = true
	}

	// VALIDASI STOK
	if newQuantity > stock {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"message": "Quantity exceeds available stock!"})
		return
	}

	grossAmount := float64(newQuantity) * price
	currentTime := time.Now()

	if isUpdate {
		// UPDATE Keranjang Lama
		_, err = config.DB.Exec("UPDATE carts SET quantity = ?, gross_amount = ?, updated_at = ? WHERE id = ?", newQuantity, grossAmount, currentTime, cartID)
	} else {
		// INSERT Keranjang Baru
		result, errInsert := config.DB.Exec("INSERT INTO carts (user_id, product_id, quantity, gross_amount, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			userID, req.ProductID, newQuantity, grossAmount, currentTime, currentTime)
		err = errInsert

		if err == nil {
			id, _ := result.LastInsertId()
			cartID = int(id)
		}
	}

	if err != nil {
		http.Error(w, "Failed to save cart", http.StatusInternalServerError)
		return
	}

	// Kembalikan Response persis seperti Laravel Anda
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Added to cart successfully",
		"cart_id": cartID,
	})
}

// =========================================================================
// 3. PUT: Update Kuantitas (update)
// =========================================================================
func UpdateCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cartID := r.PathValue("id")

	var req CartRequest
	json.NewDecoder(r.Body).Decode(&req)

	// Validasi kepemilikan keranjang dan ambil product_id
	var productID int
	err = config.DB.QueryRow("SELECT product_id FROM carts WHERE id = ? AND user_id = ?", cartID, userID).Scan(&productID)
	if err == sql.ErrNoRows {
		http.Error(w, "Cart item not found or unauthorized", http.StatusNotFound)
		return
	}

	// Cek Stok dan Harga Terbaru
	var stock int
	var price float64
	config.DB.QueryRow("SELECT stock, price FROM products WHERE id = ?", productID).Scan(&stock, &price)

	if req.Quantity > stock {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"message": "Stock limited!"})
		return
	}

	grossAmount := float64(req.Quantity) * price
	_, err = config.DB.Exec("UPDATE carts SET quantity = ?, gross_amount = ?, updated_at = ? WHERE id = ?", req.Quantity, grossAmount, time.Now(), cartID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Cart updated successfully"})
}

// =========================================================================
// 4. DELETE: Hapus dari Keranjang (destroy)
// =========================================================================
func DeleteCart(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cartID := r.PathValue("id")

	// Pastikan user hanya bisa menghapus keranjangnya sendiri
	_, err = config.DB.Exec("DELETE FROM carts WHERE id = ? AND user_id = ?", cartID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Item removed"})
}
