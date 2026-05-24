package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/seanalden-great/gycora-api/config"

)

// =========================================================================
// STRUKTUR DATA (Meniru FormRequest & Resource Laravel)
// =========================================================================

// AddressRequest menampung input JSON dari frontend
type AddressRequest struct {
	Region           string `json:"region"`
	FirstNameAddress string `json:"first_name_address"`
	LastNameAddress  string `json:"last_name_address"`
	AddressLocation  string `json:"address_location"`
	City             string `json:"city"`
	Province         string `json:"province"`
	PostalCode       string `json:"postal_code"`
	LocationType     string `json:"location_type"`
	Latitude         string `json:"latitude"`
	Longitude        string `json:"longitude"`
	IsDefault        bool   `json:"is_default"`
}

// Struct untuk memformat output JSON (Meniru AddressResource.php)
type AddressReceiver struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
}

type AddressDetails struct {
	Region     string `json:"region"`
	Location   string `json:"address_location"` // di Laravel namanya 'location', tapi sesuaikan jika perlu
	Type       string `json:"type"`
	City       string `json:"city"`
	Province   string `json:"province"`
	PostalCode string `json:"postal_code"`
	Latitude   string `json:"latitude"`
	Longitude  string `json:"longitude"`
}

type AddressResource struct {
	ID        int             `json:"id"`
	Receiver  AddressReceiver `json:"receiver"`
	Details   AddressDetails  `json:"details"`
	IsDefault bool            `json:"is_default"`
	CreatedAt string          `json:"created_at"`
}

// =========================================================================
// 1. GET: Ambil Semua Alamat (index)
// =========================================================================
func GetAddresses(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r) // Fungsi dari cart_ctrl.go
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `
		SELECT id, region, first_name_address, last_name_address, address_location,
		       location_type, city, province, postal_code, latitude, longitude, is_default, created_at
		FROM addresses
		WHERE user_id = ?
		ORDER BY id DESC
	`

	rows, err := config.DB.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var addresses []AddressResource

	for rows.Next() {
		var id int
		var region, fName, lName, addressLoc, city, prov, postal string
		var locType, lat, lng sql.NullString
		var isDefault bool
		var createdAt []byte

		err := rows.Scan(&id, &region, &fName, &lName, &addressLoc, &locType, &city, &prov, &postal, &lat, &lng, &isDefault, &createdAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Handle Nilai Null dari DB (Meniru logic coalescing di PHP)
		finalLocType := "other"
		if locType.Valid && locType.String != "" {
			finalLocType = locType.String
		}

		// Membangun format resource
		addrRes := AddressResource{
			ID: id,
			Receiver: AddressReceiver{
				FirstName: fName,
				LastName:  lName,
				FullName:  fName + " " + lName,
			},
			Details: AddressDetails{
				Region:     region,
				Location:   addressLoc,
				Type:       finalLocType,
				City:       city,
				Province:   prov,
				PostalCode: postal,
				Latitude:   lat.String,
				Longitude:  lng.String,
			},
			IsDefault: isDefault,
			CreatedAt: string(createdAt),
		}

		addresses = append(addresses, addrRes)
	}

	if len(addresses) == 0 {
		addresses = []AddressResource{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addresses)
}

// // =========================================================================
// // 2. POST: Tambah Alamat (store) - DENGAN DB TRANSACTION
// // =========================================================================
// func CreateAddress(w http.ResponseWriter, r *http.Request) {
// 	userID, err := getUserIDFromToken(r)
// 	if err != nil {
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	var req AddressRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid payload", http.StatusBadRequest)
// 		return
// 	}

// 	// Memulai Database Transaction (Pengganti DB::transaction di Laravel)
// 	tx, err := config.DB.Begin()
// 	if err != nil {
// 		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
// 		return
// 	}

// 	// Logic resetDefaultAddress
// 	// if req.IsDefault {
// 	// 	_, err = tx.Exec("UPDATE addresses SET is_default = 0 WHERE user_id = ?", userID)
// 	// 	if err != nil {
// 	// 		tx.Rollback()
// 	// 		http.Error(w, "Failed to reset default addresses", http.StatusInternalServerError)
// 	// 		return
// 	// 	}
// 	// }

// 	// Logic resetDefaultAddress
// 	if req.IsDefault {
// 		// Gunakan tx.Exec, dan tangkap err
// 		_, err = tx.Exec("UPDATE addresses SET is_default = 0 WHERE user_id = ?", userID)
// 		if err != nil {
// 			tx.Rollback()
// 			// [PERBAIKAN] Tampilkan err.Error() agar kita tahu apa keluhan asli MySQL
// 			http.Error(w, "Failed to reset default addresses: "+err.Error(), http.StatusInternalServerError)
// 			return
// 		}
// 	}

// 	currentTime := time.Now().Format("2006-01-02 15:04:05")

// 	query := `
// 		INSERT INTO addresses (user_id, region, first_name_address, last_name_address, address_location,
// 		location_type, city, province, postal_code, latitude, longitude, is_default, created_at, updated_at)
// 		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
// 	`
// 	result, err := tx.Exec(query, userID, req.Region, req.FirstNameAddress, req.LastNameAddress, req.AddressLocation,
// 		req.LocationType, req.City, req.Province, req.PostalCode, req.Latitude, req.Longitude, req.IsDefault, currentTime, currentTime)

// 	if err != nil {
// 		tx.Rollback()
// 		http.Error(w, "Failed to insert address: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	// Commit transaksi jika semua berhasil
// 	tx.Commit()

// 	// Menyiapkan respons
// 	id, _ := result.LastInsertId()

// 	// Format ulang ke Resource (Meniru return new AddressResource)
// 	response := AddressResource{
// 		ID: int(id),
// 		Receiver: AddressReceiver{
// 			FirstName: req.FirstNameAddress,
// 			LastName:  req.LastNameAddress,
// 			FullName:  req.FirstNameAddress + " " + req.LastNameAddress,
// 		},
// 		Details: AddressDetails{
// 			Region:     req.Region,
// 			Location:   req.AddressLocation,
// 			Type:       req.LocationType,
// 			City:       req.City,
// 			Province:   req.Province,
// 			PostalCode: req.PostalCode,
// 			Latitude:   req.Latitude,
// 			Longitude:  req.Longitude,
// 		},
// 		IsDefault: req.IsDefault,
// 		CreatedAt: currentTime,
// 	}

// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(response)
// }

// // =========================================================================
// // 3. PUT: Update Alamat (update)
// // =========================================================================
// func UpdateAddress(w http.ResponseWriter, r *http.Request) {
// 	userID, err := getUserIDFromToken(r)
// 	if err != nil {
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	addressID := r.PathValue("id")

// 	// Pastikan alamat tersebut milik user yang sedang login
// 	var exists int
// 	err = config.DB.QueryRow("SELECT id FROM addresses WHERE id = ? AND user_id = ?", addressID, userID).Scan(&exists)
// 	if err == sql.ErrNoRows {
// 		http.Error(w, "Address not found or unauthorized", http.StatusNotFound)
// 		return
// 	}

// 	var req AddressRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid payload", http.StatusBadRequest)
// 		return
// 	}

// 	tx, err := config.DB.Begin()
// 	if err != nil {
// 		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
// 		return
// 	}

// 	if req.IsDefault {
// 		_, err = tx.Exec("UPDATE addresses SET is_default = 0 WHERE user_id = ? AND id != ?", userID, addressID)
// 		if err != nil {
// 			tx.Rollback()
// 			http.Error(w, "Failed to reset default addresses", http.StatusInternalServerError)
// 			return
// 		}
// 	}

// 	currentTime := time.Now().Format("2006-01-02 15:04:05")

// 	query := `
// 		UPDATE addresses SET region=?, first_name_address=?, last_name_address=?, address_location=?,
// 		location_type=?, city=?, province=?, postal_code=?, latitude=?, longitude=?, is_default=?, updated_at=?
// 		WHERE id = ? AND user_id = ?
// 	`
// 	_, err = tx.Exec(query, req.Region, req.FirstNameAddress, req.LastNameAddress, req.AddressLocation,
// 		req.LocationType, req.City, req.Province, req.PostalCode, req.Latitude, req.Longitude, req.IsDefault, currentTime, addressID, userID)

// 	if err != nil {
// 		tx.Rollback()
// 		http.Error(w, "Failed to update address", http.StatusInternalServerError)
// 		return
// 	}

// 	tx.Commit()

// 	// Cukup kembalikan pesan sukses untuk mempercepat respon jaringan
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]string{"message": "Address updated successfully"})
// }

// =========================================================================
// 2. POST: Tambah Alamat (store)
// =========================================================================
func CreateAddress(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req AddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	tx, err := config.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	// [PERBAIKAN] Casting boolean Golang ke Integer MySQL (0 atau 1)
	isDefaultInt := 0
	if req.IsDefault {
		isDefaultInt = 1
		// Reset alamat lain milik user ini menjadi 0
		_, err = tx.Exec("UPDATE addresses SET is_default = 0 WHERE user_id = ?", userID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to reset default addresses: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	currentTime := time.Now().Format("2006-01-02 15:04:05")

	// Pastikan kita memasukkan isDefaultInt (integer), bukan req.IsDefault (boolean)
	query := `
		INSERT INTO addresses (user_id, region, first_name_address, last_name_address, address_location,
		location_type, city, province, postal_code, latitude, longitude, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := tx.Exec(query, userID, req.Region, req.FirstNameAddress, req.LastNameAddress, req.AddressLocation,
		req.LocationType, req.City, req.Province, req.PostalCode, req.Latitude, req.Longitude, isDefaultInt, currentTime, currentTime)

	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to insert address: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tx.Commit()

	// ... (Sisa kode pembuatan respons JSON tetap sama seperti sebelumnya) ...
	id, _ := result.LastInsertId()
	response := AddressResource{
		ID: int(id),
		Receiver: AddressReceiver{
			FirstName: req.FirstNameAddress,
			LastName:  req.LastNameAddress,
			FullName:  req.FirstNameAddress + " " + req.LastNameAddress,
		},
		Details: AddressDetails{
			Region:     req.Region,
			Location:   req.AddressLocation,
			Type:       req.LocationType,
			City:       req.City,
			Province:   req.Province,
			PostalCode: req.PostalCode,
			Latitude:   req.Latitude,
			Longitude:  req.Longitude,
		},
		IsDefault: req.IsDefault,
		CreatedAt: currentTime,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// =========================================================================
// 3. PUT: Update Alamat (update)
// =========================================================================
func UpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addressID := r.PathValue("id")

	var exists int
	err = config.DB.QueryRow("SELECT id FROM addresses WHERE id = ? AND user_id = ?", addressID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		http.Error(w, "Address not found", http.StatusNotFound)
		return
	}

	var req AddressRequest
	json.NewDecoder(r.Body).Decode(&req)

	tx, err := config.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}

	// [PERBAIKAN] Casting boolean ke Integer
	isDefaultInt := 0
	if req.IsDefault {
		isDefaultInt = 1
		_, err = tx.Exec("UPDATE addresses SET is_default = 0 WHERE user_id = ? AND id != ?", userID, addressID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to reset default addresses: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	currentTime := time.Now().Format("2006-01-02 15:04:05")

	// Pastikan kita memakai isDefaultInt
	query := `
		UPDATE addresses SET region=?, first_name_address=?, last_name_address=?, address_location=?,
		location_type=?, city=?, province=?, postal_code=?, latitude=?, longitude=?, is_default=?, updated_at=?
		WHERE id = ? AND user_id = ?
	`
	_, err = tx.Exec(query, req.Region, req.FirstNameAddress, req.LastNameAddress, req.AddressLocation,
		req.LocationType, req.City, req.Province, req.PostalCode, req.Latitude, req.Longitude, isDefaultInt, currentTime, addressID, userID)

	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to update address: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tx.Commit()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Address updated successfully"})
}

// =========================================================================
// 4. DELETE: Hapus Alamat (destroy)
// =========================================================================
func DeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromToken(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addressID := r.PathValue("id")

	// Delete dengan pengamanan (hanya bisa hapus alamat sendiri)
	_, err = config.DB.Exec("DELETE FROM addresses WHERE id = ? AND user_id = ?", addressID, userID)
	if err != nil {
		http.Error(w, "Failed to delete address", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Address successfully deleted"})
}
