package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// This file contains a http.HandleFunc wrapper to always return a success or error.
// The "success" and "error" responses are defined in the "HandlerSuccess" and "HandlerError" structs
// and can be used as json responses.
// See indexHandler.go for an example
type ApiHandlerFunc func(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError)

type ApiMiddlewareFunc func(ApiHandlerFunc) ApiHandlerFunc

type HandlerSuccess struct {
	Status int `json:"-"`
	Data   interface{}
}

type HandlerError struct {
	Status  int `json:"-"`
	Message ErrorResponse
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

// This function is a http.HandlerFunc adapter for my custom HandlerFunc called ApiHandlerFunc.
func ApiHandlerAdapter(handler ApiHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		success, err := handler(w, r)

		if err != nil {
			w.WriteHeader(err.Status)
			json.NewEncoder(w).Encode(err.Message)
			return
		}

		if success != nil {
			w.WriteHeader(success.Status)
			json.NewEncoder(w).Encode(success.Data)
		}
	}
}

// The adapter that turns it into a Chi middleware
func MiddlewareAdapter(mw ApiMiddlewareFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Convert http.Handler to your ApiHandlerFunc
		handler := func(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
			// This "fake" ApiHandlerFunc just calls the next handler
			next.ServeHTTP(w, r)
			return nil, nil
		}

		// Wrap it with your middleware
		wrapped := mw(handler)

		// Return a standard http.HandlerFunc that calls your middleware-wrapped handler
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			success, err := wrapped(w, r)

			if err != nil {
				w.WriteHeader(err.Status)
				_ = json.NewEncoder(w).Encode(err.Message)
				return
			}

			if success != nil {
				w.WriteHeader(success.Status)
				if success.Data != nil {
					_ = json.NewEncoder(w).Encode(success.Data)
				}
			}
		})
	}
}

// This function verifies a JWT token and it will be used by many handlers
func VerifyJwtToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		log.Printf("[APIHandler:VerifyJwtToken] Error verifying JWT token: %v", err)
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		log.Printf("[APIHandler:VerifyJwtToken] Successfully verified JWT token: %v", claims)
		return claims, nil
	} else {
		log.Printf("[APIHandler:VerifyJwtToken] Error verifying JWT token: %v", err)
		return nil, err
	}
}
