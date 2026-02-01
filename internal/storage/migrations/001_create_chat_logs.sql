-- +goose Up
CREATE TYPE answer_type AS ENUM ('grounded', 'no_answer');

CREATE TABLE chat_logs (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  question_redacted text NOT NULL,
  question_hash text NOT NULL,
  answer_type answer_type NOT NULL,
  top_sources text[] NOT NULL DEFAULT '{}',
  top_scores real[] NOT NULL DEFAULT '{}',
  latency_ms integer NOT NULL
);

CREATE INDEX chat_logs_created_at_idx ON chat_logs (created_at DESC);
CREATE INDEX chat_logs_question_hash_idx ON chat_logs (question_hash);

-- +goose Down
DROP TABLE chat_logs;
DROP TYPE answer_type;
