package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/RafaelEstevam/go-rocket/internal/store/pgstore"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
)

type apiHandler struct {
	query       *pgstore.Queries
	router      *chi.Mux
	upgrader    websocket.Upgrader
	subscribers map[string]map[*websocket.Conn]context.CancelFunc
	mutex       *sync.Mutex
}

func (handler apiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.router.ServeHTTP(writer, request)
}

func NewHandler(query *pgstore.Queries) http.Handler {
	api := apiHandler{
		query:       query,
		upgrader:    websocket.Upgrader{CheckOrigin: func(route *http.Request) bool { return true }},
		subscribers: make(map[string]map[*websocket.Conn]context.CancelFunc),
		mutex:       &sync.Mutex{},
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID, middleware.Recoverer, middleware.Logger)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Get("/subscribe/{room_id}", api.handleSubscribe)

	router.Route("/api", func(route chi.Router) {
		router.Route("/rooms", func(route chi.Router) {
			router.Post("/", api.handlerCreateRoom)
			router.Get("/", api.handlerGetRoom)

			router.Route("/{room_id}/messages", func(route chi.Router) {
				router.Post("/", api.handlerCreateRoomMessages)
				router.Get("/", api.handlerGetRoomMessagesById)

				route.Route("/{message_id}", func(route chi.Router) {
					router.Get("/", api.handlerGetRoomMessageById)
					router.Patch("/react", api.handlerReactToMessageById)
					router.Delete("/react", api.handlerRemoveReactToMessageById)
					router.Patch("/answer", api.handlerMarkMessageAnsweredById)
				})
			})

		})
	})

	api.router = router
	return api
}

func (h apiHandler) handleSubscribe(writer http.ResponseWriter, request *http.Request) {
	rawRoomId := chi.URLParam(request, "room_id")
	roomId, err := uuid.Parse(rawRoomId)

	if err != nil {
		http.Error(writer, "Invalid room Id", http.StatusBadRequest)
		return
	}

	_, err = h.query.GetRoom(request.Context(), roomId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(writer, "Room not found", http.StatusBadRequest)
			return
		}
		http.Error(writer, "Something went wrong", http.StatusInternalServerError)
		return
	}

	connection, err := h.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		slog.Warn("Failed to upgrade connection", "error", err)
		http.Error(writer, "Failed to upgrade to websocket connection", http.StatusBadRequest)
		return
	}

	defer connection.Close()

	ctx, cancel := context.WithCancel(request.Context())

	h.mutex.Lock()
	if _, ok := h.subscribers[rawRoomId]; !ok {
		h.subscribers[rawRoomId] = make(map[*websocket.Conn]context.CancelFunc)
	}
	slog.Info("new client connected", "roow_id", rawRoomId, "IP", request.RemoteAddr)
	h.subscribers[rawRoomId][connection] = cancel
	h.mutex.Unlock()

	<-ctx.Done()

	h.mutex.Lock()
	delete(h.subscribers[rawRoomId], connection)
	h.mutex.Unlock()

}

func (h apiHandler) handlerCreateRoom(writer http.ResponseWriter, request *http.Request) {
	type _body struct {
		Theme string `json:"theme"`
	}
	var body _body
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "Invalid json", http.StatusBadRequest)
		return
	}

	roomId, err := h.query.InsertRoom(request.Context(), body.Theme)
	if err != nil {
		slog.Error("Failwd to insert room", "error", err)
		http.Error(writer, "Something went wrong", http.StatusInternalServerError)
		return
	}

	type response struct {
		ID string `json:"theme"`
	}

	data, _ := json.Marshal(response{ID: roomId.String()})

	writer.Header().Set("Content-type", "application/json")
	res, err := writer.Write(data)

	slog.Error("", res)

	if err != nil {
		slog.Error("Failwd to insert room", "error", err)
	}

}
func (h apiHandler) handlerGetRoom(writer http.ResponseWriter, request *http.Request)             {}
func (h apiHandler) handlerGetRoomMessagesById(writer http.ResponseWriter, request *http.Request) {}
func (h apiHandler) handlerCreateRoomMessages(writer http.ResponseWriter, request *http.Request)  {}
func (h apiHandler) handlerGetRoomMessageById(writer http.ResponseWriter, request *http.Request)  {}
func (h apiHandler) handlerReactToMessageById(writer http.ResponseWriter, request *http.Request)  {}
func (h apiHandler) handlerMarkMessageAnsweredById(writer http.ResponseWriter, request *http.Request) {
}
func (h apiHandler) handlerRemoveReactToMessageById(writer http.ResponseWriter, request *http.Request) {
}

func helloWorldHandler(w http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}
