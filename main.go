// package main

// import (
// 	"fmt"
// 	"log"
// 	"net/http"

// 	"github.com/seanalden-great/gycora-api/config"
// 	"github.com/seanalden-great/gycora-api/routes" // Import folder routes baru

// )

// func main() {
// 	config.ConnectDB()
// 	fmt.Println("Database Connected!")

// 	mux := http.NewServeMux()

// 	// Panggil semua rute dari file terpisah (seperti include api.php)
// 	routes.RegisterRoutes(mux)

// 	handler := corsMiddleware(mux)

// 	fileServer := http.FileServer(http.Dir("./uploads"))
// 	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fileServer))
// 	fmt.Println("Server running on port 8080")
// 	log.Fatal(http.ListenAndServe(":8080", handler))
// }

// func corsMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Access-Control-Allow-Origin", "*")
// 		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
// 		// w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

// 		// [PERBAIKAN] Tambahkan "Authorization" agar Golang mau menerima token JWT
// 		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
// 		if r.Method == "OPTIONS" {
// 			w.WriteHeader(http.StatusOK)
// 			return
// 		}
// 		next.ServeHTTP(w, r)
// 	})
// }

package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/SeanAlden/my-store/config"
	"github.com/SeanAlden/my-store/routes"
	"github.com/go-chi/chi/v5"
)

func main() {
	config.ConnectDB()
	fmt.Println("Database Connected!")

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This API Running Successfully"))
	})

	// routes API
	routes.RegisterRoutes(r)

	// uploads static file
	fileServer := http.FileServer(http.Dir("./uploads"))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", fileServer))

	// wrap middleware
	handler := corsMiddleware(r)

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// [PERBAIKAN] Tambahkan "Authorization" agar Golang mau menerima token JWT
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
