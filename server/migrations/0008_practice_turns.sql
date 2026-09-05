ALTER TABLE practice_sessions
    ADD COLUMN current_question_id uuid REFERENCES practice_questions(id) ON DELETE RESTRICT;

CREATE INDEX practice_sessions_current_question_idx ON practice_sessions (current_question_id);

CREATE TABLE practice_turns (
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES practice_sessions(id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES practice_questions(id) ON DELETE RESTRICT,
    actor_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content text NOT NULL CHECK (btrim(content) <> ''),
    created_at timestamptz NOT NULL,
    UNIQUE (session_id, question_id)
);

CREATE INDEX practice_turns_session_created_idx ON practice_turns (session_id, created_at ASC);
