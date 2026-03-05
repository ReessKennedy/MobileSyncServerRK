package models

type PushEvent struct {
    EventID    string                 `json:"event_id"`
    EntityType string                 `json:"entity_type"`
    EntityID   string                 `json:"entity_id"`
    Op         string                 `json:"op"`
    Payload    map[string]any         `json:"payload"`
}

type PushRequest struct {
    ClientID string      `json:"client_id"`
    Events   []PushEvent `json:"events"`
}

type Change struct {
    ID         uint64                 `db:"id" json:"id"`
    EntityType string                 `db:"entity_type" json:"entity_type"`
    EntityID   string                 `db:"entity_id" json:"entity_id"`
    Op         string                 `db:"op" json:"op"`
    Payload    map[string]any         `db:"payload_json" json:"payload"`
}

type PullResponse struct {
    NextCursor uint64   `json:"next_cursor"`
    Changes    []Change `json:"changes"`
}
