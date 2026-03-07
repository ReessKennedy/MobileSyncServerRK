-- Notes table
CREATE TABLE IF NOT EXISTS notes (
  id CHAR(36) PRIMARY KEY,
  title TEXT NULL,
  body TEXT NULL,
  type VARCHAR(16) NOT NULL,
  text TEXT NULL,
  audio_file_name TEXT NULL,
  audio_duration DOUBLE NULL,
  photo_file_name TEXT NULL,
  is_completed BOOLEAN NOT NULL DEFAULT FALSE,
  transcription TEXT NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  deleted_at DATETIME(6) NULL,
  server_id CHAR(36) NULL,
  public_id CHAR(11) NULL,
  version INT NULL,
  client_id CHAR(36) NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS notes_public_id_idx ON notes(public_id);

-- Changes table (cursor)
CREATE TABLE IF NOT EXISTS changes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  entity_type VARCHAR(50) NOT NULL,
  entity_id CHAR(36) NOT NULL,
  op ENUM('upsert','delete') NOT NULL,
  payload_json JSON NOT NULL,
  server_ts TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  INDEX idx_id (id),
  INDEX idx_entity (entity_type, entity_id)
);

-- Idempotency table
CREATE TABLE IF NOT EXISTS seen_events (
  event_id CHAR(36) PRIMARY KEY,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);
