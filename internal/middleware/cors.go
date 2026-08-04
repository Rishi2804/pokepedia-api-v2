package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// CORS allows the frontend dev server to call this API from a different
// origin. Adjust AllowedOrigins for staging/prod once those exist.
func CORS() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:1513"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
