package api

import (
	"net/http"

	"github.com/RafaelEstevam/go-rocket/internal/store/pgstore"
	"github.com/go-chi/chi/v5"
)

type apiHandler struct {
	query  *pgstore.Queries
	router *chi.Mux
}

func (handler apiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.router.ServeHTTP(writer, request)
}

func NewHandler(query *pgstore.Queries) http.Handler {
	api := apiHandler{
		query: query,
	}

	router := chi.NewRouter()

	api.router = router

	return api
}
