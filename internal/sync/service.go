package syncsvc

import (
    "database/sql"
    "encoding/json"
    "errors"

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
                _, err = tx.Exec(`
                    INSERT INTO notes (
                        id, type, text, audio_file_name, audio_duration, photo_file_name,
                        is_completed, transcription, created_at, updated_at, deleted_at, server_id, version, client_id
                    ) VALUES (
                        :id, :type, :text, :audio_file_name, :audio_duration, :photo_file_name,
                        :is_completed, :transcription, :created_at, :updated_at, :deleted_at, :server_id, :version, :client_id
                    )
                    ON DUPLICATE KEY UPDATE
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
                `, ev.Payload)
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
            ID: r.ID,
            EntityType: r.EntityType,
            EntityID: r.EntityID,
            Op: r.Op,
            Payload: payload,
        })
        nextCursor = r.ID
    }

    return changes, nextCursor, nil
}
