package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "mobilesyncserverrk/internal/models"
    syncsvc "mobilesyncserverrk/internal/sync"
)

type SyncHandler struct {
    Service *syncsvc.Service
}

func (h *SyncHandler) Push(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req models.PushRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    if err := h.Service.Push(req); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *SyncHandler) Pull(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    cursorStr := r.URL.Query().Get("cursor")
    limitStr := r.URL.Query().Get("limit")

    var cursor uint64
    if cursorStr != "" {
        if v, err := strconv.ParseUint(cursorStr, 10, 64); err == nil {
            cursor = v
        }
    }

    limit := 500
    if limitStr != "" {
        if v, err := strconv.Atoi(limitStr); err == nil {
            limit = v
        }
    }

    changes, nextCursor, err := h.Service.Pull(cursor, limit)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    resp := models.PullResponse{
        NextCursor: nextCursor,
        Changes: changes,
    }

    w.WriteHeader(http.StatusOK)
    _ = json.NewEncoder(w).Encode(resp)
}
