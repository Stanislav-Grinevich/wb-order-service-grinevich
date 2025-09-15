// Package httpserver поднимает HTTP API и раздаёт статику.
package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"

	"github.com/go-chi/chi/v5"
)

// Server инкапсулирует кэш, репозиторий и роутер.
type Server struct {
	cache *cache.Cache
	repo  *repo.OrdersRepo
	mux   *chi.Mux
}

// New создаёт сервер и настраивает маршруты.
func New(c *cache.Cache, r *repo.OrdersRepo) *Server {
	s := &Server{
		cache: c,
		repo:  r,
		mux:   chi.NewRouter(),
	}
	s.routes()
	return s
}

// routes описывает эндпоинты.
func (s *Server) routes() {
	r := s.mux

	// проверка работоспособности
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// получение заказа по ID
	r.Get("/order/{id}", s.handleGetOrder)

	// статика
	fs := http.FileServer(http.Dir("web"))
	r.Handle("/*", fs)
}

// handleGetOrder ищет заказ в кэше или БД и возвращает в JSON.
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}

	if o, ok := s.cache.Get(id); ok {
		writeJSON(w, o)
		return
	}

	o, err := s.repo.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	s.cache.Set(o)
	writeJSON(w, o)
}

// writeJSON возвращает объект в JSON с отступами.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// Handler возвращает объект http.Handler для запуска сервера.
func (s *Server) Handler() http.Handler {
	return s.mux
}
