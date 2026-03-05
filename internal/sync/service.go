package syncsvc

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"mobilesyncserverrk/internal/models"
)

type Service struct {
	DB *sqlx.DB
}

func (s *Service) Push(req models.PushRequest) error {
	tx, err := s.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, ev := range req.Events {
		// Idempotency check
		var exists string
		err = tx.Get(&exists, "SELECT event_id FROM seen_events WHERE event_id = ?", ev.EventID)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		if _, err := tx.Exec("INSERT INTO seen_events(event_id) VALUES (?)", ev.EventID); err != nil {
			return err
		}

		payloadJSON, err := json.Marshal(ev.Payload)
		if err != nil {
			return err
		}

		// Apply to notes (canonical)
		if ev.EntityType == "note" {
			if ev.Op == "delete" {
				_, err = tx.Exec(`UPDATE notes SET deleted_at = NOW(6), updated_at = NOW(6) WHERE id = ?`, ev.EntityID)
			} else {
				params := normalizeNotePayload(ev.Payload, ev.EntityID)
				_, err = tx.NamedExec(`
                    INSERT INTO notes (
                        id, title, body, type, text, audio_file_name, audio_duration, photo_file_name,
                        is_completed, transcription, created_at, updated_at, deleted_at, server_id, version, client_id
                    ) VALUES (
                        :id, :title, :body, :type, :text, :audio_file_name, :audio_duration, :photo_file_name,
                        :is_completed, :transcription, :created_at, :updated_at, :deleted_at, :server_id, :version, :client_id
                    )
                    ON DUPLICATE KEY UPDATE
                        title = VALUES(title),
                        body = VALUES(body),
                        type = VALUES(type),
                        text = VALUES(text),
                        audio_file_name = VALUES(audio_file_name),
                        audio_duration = VALUES(audio_duration),
                        photo_file_name = VALUES(photo_file_name),
                        is_completed = VALUES(is_completed),
                        transcription = VALUES(transcription),
                        created_at = VALUES(created_at),
                        updated_at = VALUES(updated_at),
                        deleted_at = VALUES(deleted_at),
                        server_id = VALUES(server_id),
                        version = VALUES(version),
                        client_id = VALUES(client_id)
                `, params)
			}
			if err != nil {
				return err
			}
		}

		// Append to changes log
		if _, err := tx.Exec(`INSERT INTO changes(entity_type, entity_id, op, payload_json) VALUES (?, ?, ?, ?)`,
			ev.EntityType, ev.EntityID, ev.Op, payloadJSON); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func normalizeNotePayload(payload map[string]any, fallbackID string) map[string]any {
	now := time.Now().UTC()
	params := map[string]any{}

	// ID
	if v, ok := payload["id"]; ok {
		params["id"] = v
	} else {
		params["id"] = fallbackID
	}

	// Title/body (optional, may be absent in mobile client)
	if v, ok := payload["title"]; ok {
		params["title"] = v
	} else {
		params["title"] = nil
	}
	if v, ok := payload["body"]; ok {
		params["body"] = v
	} else {
		params["body"] = nil
	}

	// Strings
	params["type"] = first(payload, "type")
	params["text"] = first(payload, "text")
	params["audio_file_name"] = first(payload, "audio_file_name", "audioFileName")
	params["photo_file_name"] = first(payload, "photo_file_name", "photoFileName")
	params["transcription"] = first(payload, "transcription")

	// Numbers
	params["audio_duration"] = first(payload, "audio_duration", "audioDuration")
	params["version"] = first(payload, "version")

	// Bools
	params["is_completed"] = boolOr(payload, false, "is_completed", "isCompleted")

	// IDs
	params["server_id"] = first(payload, "server_id", "serverId")
	params["client_id"] = first(payload, "client_id", "clientId")

	// Timestamps: accept RFC3339 strings or epoch seconds
	params["created_at"] = timeOr(payload, now, "created_at", "createdAt")
	params["updated_at"] = timeOr(payload, now, "updated_at", "updatedAt")
	if t, ok := timeOrNullable(payload, "deleted_at", "deletedAt"); ok {
		params["deleted_at"] = t
	} else {
		params["deleted_at"] = nil
	}

	return params
}

func first(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func boolOr(m map[string]any, def bool, keys ...string) bool {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
			if f, ok := v.(float64); ok {
				return f != 0
			}
		}
	}
	return def
}

func timeOr(m map[string]any, def time.Time, keys ...string) time.Time {
	if t, ok := timeOrNullable(m, keys...); ok {
		return t
	}
	return def
}

func timeOrNullable(m map[string]any, keys ...string) (time.Time, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch val := v.(type) {
			case string:
				if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
					return t, true
				}
			case float64:
				sec := int64(val)
				nsec := int64((val - float64(sec)) * 1e9)
				return time.Unix(sec, nsec).UTC(), true
			case json.Number:
				if f, err := val.Float64(); err == nil {
					sec := int64(f)
					nsec := int64((f - float64(sec)) * 1e9)
					return time.Unix(sec, nsec).UTC(), true
				}
			case time.Time:
				return val, true
			}
		}
	}
	return time.Time{}, false
}

func (s *Service) Pull(cursor uint64, limit int) ([]models.Change, uint64, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	rows := []struct {
		ID         uint64 `db:"id"`
		EntityType string `db:"entity_type"`
		EntityID   string `db:"entity_id"`
		Op         string `db:"op"`
		Payload    []byte `db:"payload_json"`
	}{}

	err := s.DB.Select(&rows, `
        SELECT id, entity_type, entity_id, op, payload_json
        FROM changes
        WHERE id > ?
        ORDER BY id ASC
        LIMIT ?
    `, cursor, limit)
	if err != nil {
		return nil, cursor, err
	}

	changes := make([]models.Change, 0, len(rows))
	var nextCursor = cursor
	for _, r := range rows {
		var payload map[string]any
		_ = json.Unmarshal(r.Payload, &payload)
		changes = append(changes, models.Change{
			ID:         r.ID,
			EntityType: r.EntityType,
			EntityID:   r.EntityID,
			Op:         r.Op,
			Payload:    payload,
		})
		nextCursor = r.ID
	}

	return changes, nextCursor, nil
}
