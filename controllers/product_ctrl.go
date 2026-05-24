package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/SeanAlden/my-store/config"

)

// Struktur Data Produk
type Product struct {
	ID           int     `json:"id"`
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name,omitempty"` // Hasil JOIN tabel categories
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Description  string  `json:"description"`
	Benefits     string  `json:"benefits"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
	ImageURL     string  `json:"image_url"`
}

// GET: Ambil semua produk beserta nama kategorinya
func GetProducts(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT p.id, p.category_id, c.name as category_name, p.sku, p.name, p.slug, 
		       p.description, p.benefits, p.price, p.stock, p.image_url
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		ORDER BY p.id DESC
	`
	rows, err := config.DB.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var desc, benefits, imgUrl sql.NullString // Handler untuk data NULL

		err := rows.Scan(&p.ID, &p.CategoryID, &p.CategoryName, &p.SKU, &p.Name, &p.Slug,
			&desc, &benefits, &p.Price, &p.Stock, &imgUrl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		p.Description = desc.String
		p.Benefits = benefits.String
		p.ImageURL = imgUrl.String
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// GET: Ambil detail 1 produk
func GetProductByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var p Product
	var desc, benefits, imgUrl sql.NullString

	query := `SELECT id, category_id, sku, name, slug, description, benefits, price, stock, image_url FROM products WHERE id = ?`
	err := config.DB.QueryRow(query, id).Scan(&p.ID, &p.CategoryID, &p.SKU, &p.Name, &p.Slug, &desc, &benefits, &p.Price, &p.Stock, &imgUrl)
	
	if err != nil {
		http.Error(w, "Produk tidak ditemukan", http.StatusNotFound)
		return
	}

	p.Description = desc.String
	p.Benefits = benefits.String
	p.ImageURL = imgUrl.String

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Multipart Form (Max 10MB)
	r.ParseMultipartForm(10 << 20)

	// 2. Ambil Data Text
	categoryID := r.FormValue("category_id")
	sku := r.FormValue("sku")
	name := r.FormValue("name")
	slug := r.FormValue("slug")
	price := r.FormValue("price")
	stock := r.FormValue("stock")
	description := r.FormValue("description")
	benefits := r.FormValue("benefits")

	// 3. Handle Upload Gambar
	var imagePath string
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		// Buat nama file unik: timestamp + nama asli
		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(handler.Filename))
		path := filepath.Join("uploads", filename)
		
		dst, _ := os.Create(path)
		defer dst.Close()
		io.Copy(dst, file)
		imagePath = "http://localhost:8080/uploads/" + filename
	}

	// 4. Simpan ke Database
	currentTime := time.Now()
	query := `INSERT INTO products (category_id, sku, name, slug, description, benefits, price, stock, image_url, created_at, updated_at) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err = config.DB.Exec(query, categoryID, sku, name, slug, description, benefits, price, stock, imagePath, currentTime, currentTime)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Produk berhasil ditambahkan"})
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	r.ParseMultipartForm(10 << 20)

	categoryID := r.FormValue("category_id")
	sku := r.FormValue("sku")
	name := r.FormValue("name")
	slug := r.FormValue("slug")
	price := r.FormValue("price")
	stock := r.FormValue("stock")
	description := r.FormValue("description")
	benefits := r.FormValue("benefits")

	// Cek apakah ada upload gambar baru
	var imagePath string
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(handler.Filename))
		path := filepath.Join("uploads", filename)
		dst, _ := os.Create(path)
		defer dst.Close()
		io.Copy(dst, file)
		imagePath = "http://localhost:8080/uploads/" + filename
	}

	// Jika ada gambar baru, update image_url. Jika tidak, tetap gunakan yang lama.
	var query string
	var args []interface{}

	if imagePath != "" {
		query = `UPDATE products SET category_id=?, sku=?, name=?, slug=?, description=?, benefits=?, price=?, stock=?, image_url=?, updated_at=? WHERE id=?`
		args = []interface{}{categoryID, sku, name, slug, description, benefits, price, stock, imagePath, time.Now(), id}
	} else {
		query = `UPDATE products SET category_id=?, sku=?, name=?, slug=?, description=?, benefits=?, price=?, stock=?, updated_at=? WHERE id=?`
		args = []interface{}{categoryID, sku, name, slug, description, benefits, price, stock, time.Now(), id}
	}

	_, err = config.DB.Exec(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"message": "Produk diperbarui"})
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := config.DB.Exec("DELETE FROM products WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Produk berhasil dihapus"})
}