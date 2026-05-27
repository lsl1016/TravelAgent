CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id VARCHAR(128) NOT NULL UNIQUE,
  display_name VARCHAR(128) NULL,
  source VARCHAR(64) NOT NULL DEFAULT 'local',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  session_id VARCHAR(128) NOT NULL UNIQUE,
  user_id VARCHAR(128) NOT NULL,
  title VARCHAR(255) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  ended_at DATETIME(6) NULL,
  INDEX idx_sessions_user_id (user_id)
);

CREATE TABLE IF NOT EXISTS chat_messages (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  message_id VARCHAR(128) NOT NULL UNIQUE,
  session_id VARCHAR(128) NOT NULL,
  user_id VARCHAR(128) NOT NULL,
  role VARCHAR(32) NOT NULL,
  content MEDIUMTEXT NOT NULL,
  metadata JSON NULL,
  created_at DATETIME(6) NOT NULL,
  INDEX idx_messages_session_created (session_id, created_at),
  INDEX idx_messages_user_created (user_id, created_at)
);

CREATE TABLE IF NOT EXISTS preferences (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id VARCHAR(128) NOT NULL,
  type VARCHAR(128) NOT NULL,
  value_json JSON NOT NULL,
  source VARCHAR(64) NOT NULL DEFAULT 'agent',
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  UNIQUE KEY uk_preferences_user_type (user_id, type)
);

CREATE TABLE IF NOT EXISTS trip_history (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  trip_id VARCHAR(128) NOT NULL UNIQUE,
  user_id VARCHAR(128) NOT NULL,
  session_id VARCHAR(128) NULL,
  origin VARCHAR(255) NULL,
  destination VARCHAR(255) NULL,
  start_date DATE NULL,
  end_date DATE NULL,
  purpose VARCHAR(255) NULL,
  itinerary_json JSON NULL,
  raw_json JSON NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL,
  INDEX idx_trips_user_created (user_id, created_at),
  INDEX idx_trips_destination (destination)
);

CREATE TABLE IF NOT EXISTS agent_runs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  run_id VARCHAR(128) NOT NULL UNIQUE,
  user_id VARCHAR(128) NOT NULL,
  session_id VARCHAR(128) NULL,
  request_message_id VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL,
  request_json JSON NULL,
  intention_json JSON NULL,
  schedule_json JSON NULL,
  result_json JSON NULL,
  error_code VARCHAR(128) NULL,
  error_message TEXT NULL,
  duration_ms INT NULL,
  created_at DATETIME(6) NOT NULL,
  INDEX idx_agent_runs_session_created (session_id, created_at),
  INDEX idx_agent_runs_status_created (status, created_at)
);

CREATE TABLE IF NOT EXISTS user_statistics (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id VARCHAR(128) NOT NULL UNIQUE,
  total_trips INT NOT NULL DEFAULT 0,
  total_messages INT NOT NULL DEFAULT 0,
  frequent_destinations_json JSON NULL,
  created_at DATETIME(6) NOT NULL,
  updated_at DATETIME(6) NOT NULL
);
