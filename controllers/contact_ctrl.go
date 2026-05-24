package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/seanalden-great/gycora-api/config"
)

// ContactRequest hanya menerima isi pesan dari frontend
type ContactRequest struct {
	Description string `json:"description"`
}

// =========================================================================
// POST: Kirim Pesan Contact Us
// =========================================================================
func SubmitContact(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil ID user dari Token JWT
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Decode body request (hanya description)
	var req ContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.Description == "" {
		http.Error(w, "Pesan tidak boleh kosong", http.StatusUnprocessableEntity)
		return
	}

	// 3. Tarik data profil asli dari tabel users untuk menjaga integritas data
	var firstName, lastName, email string
	var phone sql.NullString // Menggunakan NullString karena phone di DB bisa NULL
	err = config.DB.QueryRow("SELECT first_name, last_name, email, phone FROM users WHERE id = ?", userID).
		Scan(&firstName, &lastName, &email, &phone)

	if err != nil {
		http.Error(w, "User tidak ditemukan", http.StatusNotFound)
		return
	}

	// Format nama lengkap dan nomor telepon
	fullName := firstName + " " + lastName
	phoneStr := ""
	if phone.Valid {
		phoneStr = phone.String
	}

	// 4. Simpan ke tabel contacts
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	query := `
		INSERT INTO contacts (user_id, name, email, phone, description, is_read, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 0, ?, ?)
	`
	_, err = config.DB.Exec(query, userID, fullName, email, phoneStr, req.Description, currentTime, currentTime)

	if err != nil {
		http.Error(w, "Gagal menyimpan pesan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Pesan Anda berhasil dikirim. Tim kami akan segera merespons.",
	})
}
