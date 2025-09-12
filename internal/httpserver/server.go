package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	cache *cache.Cache
	repo  *repo.OrdersRepo
	mux   *chi.Mux
}

func New(c *cache.Cache, r *repo.OrdersRepo) *Server {
	s := &Server{
		cache: c,
		repo:  r,
		mux:   chi.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	r := s.mux

	// health-check endpoint
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// API endpoint для получения заказа по ID
	r.Get("/order/{id}", s.handleGetOrder)

	// Раздача статики (web/index.html)
	fs := http.FileServer(http.Dir("web"))
	r.Handle("/*", fs)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}

	// Сначала пробуем взять заказ из кэша
	if o, ok := s.cache.Get(id); ok {
		writeJSON(w, o)
		return
	}

	// Если в кэше нет — идём в базу данных
	o, err := s.repo.GetOrder(r.Context(), id)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	// Кладём в кэш и отдаём клиенту
	s.cache.Set(o)
	writeJSON(w, o)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}
