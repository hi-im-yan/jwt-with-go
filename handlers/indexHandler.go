package handlers

import "net/http"

type IndexHandler struct {
}

func NewIndexHandler() *IndexHandler {
	return &IndexHandler{}
}

type healthResponse struct {
	Health string `json:"health"`
}

// @Summary Health check endpoint
// @Description Checks if the API is up and running
// @Tags index
// @Produce json
// @Success 200 {object} healthResponse
// @Router / [get]
func (ih *IndexHandler) HealthCheck(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
	return &HandlerSuccess{Status: http.StatusOK, Data: healthResponse{Health: "Alive"}}, nil
}
