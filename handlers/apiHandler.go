package handlers

import (
	"encoding/json"
	"net/http"
)

// This file contains a http.HandleFunc wrapper to always return a success or error.
// The "success" and "error" responses are defined in the "HandlerSuccess" and "HandlerError" structs
// and can be used as json responses.
// See indexHandler.go for an example
type ApiHandlerFunc func(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError)

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
