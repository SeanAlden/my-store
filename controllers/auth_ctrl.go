package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/SeanAlden/my-store/config"

)

// Gunakan secret key yang kuat di production (sebaiknya dari file .env)
var jwtSecretKey = []byte("gycora_super_secret_key_2026")

// Struktur Data untuk Request dan Response
type User struct {
	ID           int    `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Password     string `json:"-"` // "-" artinya field ini tidak akan pernah dikirim ke frontend saat di-encode ke JSON
	UserType     string `json:"usertype"`
	IsSubscribed bool   `json:"is_subscribed"`
}

type AuthRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// Struktur khusus untuk response daftar user agar lebih rapi (termasuk tanggal daftar)
type UserResponse struct {
	ID           int    `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	UserType     string `json:"usertype"`
	IsSubscribed bool   `json:"is_subscribed"`
	CreatedAt    string `json:"created_at"`
}

// Untuk update profil pengguna
type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// =========================================================================
// REGISTER USER
// =========================================================================
func Register(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validasi Dasar (Pengganti Validator::make di Laravel)
	if req.FirstName == "" || req.LastName == "" || req.Email == "" || len(req.Password) < 8 {
		http.Error(w, "Semua field wajib diisi dan password minimal 8 karakter", http.StatusUnprocessableEntity)
		return
	}

	// 1. Cek apakah email sudah ada di tabel users (unique:users)
	var existingEmail string
	err := config.DB.QueryRow("SELECT email FROM users WHERE email = ?", req.Email).Scan(&existingEmail)
	if err != sql.ErrNoRows {
		http.Error(w, "Email sudah terdaftar", http.StatusUnprocessableEntity)
		return
	}

	// 2. Cek apakah email ini sudah pernah subscribe saat menjadi Guest
	var subscriberID int
	isSubscribed := false
	err = config.DB.QueryRow("SELECT id FROM subscribers WHERE email = ?", req.Email).Scan(&subscriberID)
	if err == nil {
		isSubscribed = true
	}

	// 3. Hash Password (Pengganti Hash::make)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Gagal mengenkripsi password", http.StatusInternalServerError)
		return
	}

	// 4. Buat User baru
	currentTime := time.Now()
	result, err := config.DB.Exec(`
		INSERT INTO users (first_name, last_name, email, password, is_subscribed, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.FirstName, req.LastName, req.Email, string(hashedPassword), isSubscribed, currentTime, currentTime,
	)

	if err != nil {
		http.Error(w, "Gagal menyimpan user ke database", http.StatusInternalServerError)
		return
	}

	// 5. Jika dia ada di tabel subscribers, tandai bahwa dia kini Registered
	if isSubscribed {
		config.DB.Exec("UPDATE subscribers SET is_registered = 1 WHERE id = ?", subscriberID)
	}

	userID, _ := result.LastInsertId()
	userResponse := User{
		ID:           int(userID),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		UserType:     "user",
		IsSubscribed: isSubscribed,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User berhasil didaftarkan",
		"user":    userResponse,
	})
}

// =========================================================================
// LOGIN USER (Hanya role 'user' yang diizinkan)
// =========================================================================
func Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	json.NewDecoder(r.Body).Decode(&req)

	var user User
	var hashedPassword string

	// Ambil data user berdasarkan email
	err := config.DB.QueryRow("SELECT id, first_name, last_name, email, password, usertype FROM users WHERE email = ?", req.Email).
		Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &hashedPassword, &user.UserType)

	// Cek jika user tidak ditemukan ATAU password salah (Pengganti Hash::check) ATAU usertype BUKAN 'user'
	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)) != nil || user.UserType != "user" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Email atau Password salah."})
		return
	}

	// Generate JWT Token (Pengganti $user->createToken('auth_token')->plainTextToken)
	tokenString, err := generateJWT(user.ID, user.UserType)
	if err != nil {
		http.Error(w, "Gagal membuat token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Login Berhasil",
		"access_token": tokenString,
		"token_type":   "Bearer",
		"user":         user,
	})
}

// =========================================================================
// LOGIN ADMIN (Hanya role manajerial yang diizinkan)
// =========================================================================
func AdminLogin(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	json.NewDecoder(r.Body).Decode(&req)

	var user User
	var hashedPassword string

	// Filter khusus admin menggunakan SQL IN()
	query := `
		SELECT id, first_name, last_name, email, password, usertype 
		FROM users 
		WHERE email = ? AND usertype IN ('admin', 'superadmin', 'gudang', 'accounting')
	`
	err := config.DB.QueryRow(query, req.Email).
		Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &hashedPassword, &user.UserType)

	if err == sql.ErrNoRows || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Akses ditolak. Email/Password salah atau Anda tidak memiliki akses ke panel ini."})
		return
	}

	tokenString, err := generateJWT(user.ID, user.UserType)
	if err != nil {
		http.Error(w, "Gagal membuat token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Login Berhasil",
		"access_token": tokenString,
		"token_type":   "Bearer",
		"user":         user,
	})
}

// =========================================================================
// AMBIL SEMUA USER (Khusus Pelanggan)
// =========================================================================
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	// Query untuk mengambil user yang bukan admin/staf
	query := `
		SELECT id, first_name, last_name, email, usertype, is_subscribed, created_at 
		FROM users 
		WHERE usertype = 'user' 
		ORDER BY id DESC
	`
	
	rows, err := config.DB.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []UserResponse
	for rows.Next() {
		var u UserResponse
		var createdAt []byte // Menggunakan byte array untuk menangkap format datetime MySQL yang mentah
		
		err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.UserType, &u.IsSubscribed, &createdAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		u.CreatedAt = string(createdAt)
		users = append(users, u)
	}

	// Pastikan mengembalikan array kosong [] jika tidak ada data, bukan null
	if len(users) == 0 {
		users = []UserResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// =========================================================================
// UPDATE PROFIL USER
// =========================================================================
// func UpdateProfile(w http.ResponseWriter, r *http.Request) {
// 	userID, err := getUserIDFromToken(r) // Menggunakan helper dari cart_ctrl.go
// 	if err != nil {
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	var req UpdateProfileRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if req.FirstName == "" || req.LastName == "" || req.Email == "" {
// 		http.Error(w, "Semua field wajib diisi", http.StatusUnprocessableEntity)
// 		return
// 	}

// 	// Cek apakah email yang baru diinput sudah dipakai oleh user LAIN
// 	var existingID int
// 	err = config.DB.QueryRow("SELECT id FROM users WHERE email = ? AND id != ?", req.Email, userID).Scan(&existingID)
// 	if err != sql.ErrNoRows {
// 		w.WriteHeader(http.StatusUnprocessableEntity)
// 		json.NewEncoder(w).Encode(map[string]string{"message": "Email sudah digunakan oleh akun lain"})
// 		return
// 	}

// 	currentTime := time.Now()
// 	_, err = config.DB.Exec(`
// 		UPDATE users SET first_name = ?, last_name = ?, email = ?, updated_at = ? WHERE id = ?
// 	`, req.FirstName, req.LastName, req.Email, currentTime, userID)

// 	if err != nil {
// 		http.Error(w, "Gagal memperbarui profil", http.StatusInternalServerError)
// 		return
// 	}

// 	// Ambil data user yang baru diupdate untuk dikembalikan ke frontend
// 	var updatedUser User
// 	err = config.DB.QueryRow("SELECT id, first_name, last_name, email, usertype FROM users WHERE id = ?", userID).
// 		Scan(&updatedUser.ID, &updatedUser.FirstName, &updatedUser.LastName, &updatedUser.Email, &updatedUser.UserType)

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"message": "Profil berhasil diperbarui",
// 		"user":    updatedUser,
// 	})
// }

func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FirstName == "" || req.LastName == "" || req.Email == "" {
		http.Error(w, "Nama dan Email wajib diisi", http.StatusUnprocessableEntity)
		return
	}

	// Cek apakah email yang baru diinput sudah dipakai oleh user LAIN
	var existingID int
	err = config.DB.QueryRow("SELECT id FROM users WHERE email = ? AND id != ?", req.Email, userID).Scan(&existingID)
	if err != sql.ErrNoRows {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"message": "Email sudah digunakan oleh akun lain"})
		return
	}

	currentTime := time.Now()
	// [PERBAIKAN] Tambahkan update kolom phone
	_, err = config.DB.Exec(`
		UPDATE users SET first_name = ?, last_name = ?, email = ?, phone = ?, updated_at = ? WHERE id = ?
	`, req.FirstName, req.LastName, req.Email, req.Phone, currentTime, userID)

	if err != nil {
		http.Error(w, "Gagal memperbarui profil", http.StatusInternalServerError)
		return
	}

	// [PERBAIKAN] Ambil kembali data user termasuk phone untuk dikembalikan ke frontend
	var updatedUser User
	var phone sql.NullString // Antisipasi jika phone di DB bernilai NULL
	
	err = config.DB.QueryRow("SELECT id, first_name, last_name, email, phone, usertype FROM users WHERE id = ?", userID).
		Scan(&updatedUser.ID, &updatedUser.FirstName, &updatedUser.LastName, &updatedUser.Email, &phone, &updatedUser.UserType)

	if phone.Valid {
		updatedUser.Phone = phone.String
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Profil berhasil diperbarui",
		"user":    updatedUser,
	})
}

// Fungsi Internal untuk Generate JWT
func generateJWT(userID int, userType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   userID,
		"role":      userType,
		"exp":       time.Now().Add(time.Hour * 24).Unix(), // Token berlaku 24 Jam
	})
	return token.SignedString(jwtSecretKey)
}