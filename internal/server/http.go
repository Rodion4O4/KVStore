package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/admin/kvstore/internal/storage"
)

type KVServer struct {
	store storage.Store
}

func NewKVServer(store storage.Store) *KVServer {
	return &KVServer{store: store}
}

func (s *KVServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/set", s.handleSet)
	mux.HandleFunc("/api/v1/get", s.handleGet)
	mux.HandleFunc("/api/v1/delete", s.handleDelete)
	mux.HandleFunc("/api/v1/list", s.handleList)
	mux.HandleFunc("/api/v1/exists", s.handleExists)
	mux.HandleFunc("/health", s.handleHealth)
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, Response{Success: false, Error: message})
}

func (s *KVServer) handleSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "параметр key обязателен")
		return
	}

	sizeStr := r.URL.Query().Get("size")
	if sizeStr == "" {
		writeError(w, http.StatusBadRequest, "параметр size обязателен")
		return
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "неверный формат size")
		return
	}

	if err := s.store.Set(key, r.Body, size); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"key": key}})
}

func (s *KVServer) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "параметр key обязателен")
		return
	}

	reader, size, err := s.store.Get(key)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("X-KV-Size", strconv.FormatInt(size, 10))

	io.Copy(w, reader)
}

func (s *KVServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "параметр key обязателен")
		return
	}

	if err := s.store.Delete(key); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"key": key}})
}

func (s *KVServer) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	keys := s.store.List()
	writeJSON(w, http.StatusOK, Response{Success: true, Data: keys})
}

func (s *KVServer) handleExists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "параметр key обязателен")
		return
	}

	exists := s.store.Exists(key)
	writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]bool{"exists": exists}})
}

func (s *KVServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]string{"status": "ok"}})
}
