package controllers

import (
	"database/sql" // <-- Tambahkan baris ini
	"encoding/json"
	"net/http"
	"time"

	"github.com/SeanAlden/my-store/config"
)

type Category struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// GET: Ambil semua kategori
func GetCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := config.DB.Query("SELECT id, code, name, description, created_at FROM categories ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		var desc sql.NullString
		var createdAt sql.NullTime
		if err := rows.Scan(&cat.ID, &cat.Code, &cat.Name, &desc, &createdAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cat.Description = desc.String
		cat.CreatedAt = createdAt.Time
		categories = append(categories, cat)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// POST: Tambah kategori baru
func CreateCategory(w http.ResponseWriter, r *http.Request) {
	var cat Category
	if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentTime := time.Now()
	_, err := config.DB.Exec("INSERT INTO categories (code, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		cat.Code, cat.Name, cat.Description, currentTime, currentTime)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Kategori berhasil ditambahkan"})
}

// DELETE: Hapus kategori (Tambahan untuk kelengkapan CRUD)
func DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Fitur routing baru di Go 1.22
	
	_, err := config.DB.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Kategori berhasil dihapus"})
}

// PUT: Update kategori
func UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	
	var cat Category
	if err := json.NewDecoder(r.Body).Decode(&cat); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentTime := time.Now()
	_, err := config.DB.Exec("UPDATE categories SET code = ?, name = ?, description = ?, updated_at = ? WHERE id = ?",
		cat.Code, cat.Name, cat.Description, currentTime, id)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Kategori berhasil diperbarui"})
}