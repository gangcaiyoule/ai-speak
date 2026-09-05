CREATE TABLE practice_turns (
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES practice_sessions(id) ON DELETE CASCADE,
    question_id uuid NOT NULL REFERENCES practice_questions(id) ON DELETE RESTRICT,
    content text NOT NULL CHECK (btrim(content) <> '' AND length(content) <= 4000),
    created_at timestamptz NOT NULL,
    UNIQUE (session_id, question_id)
);

CREATE INDEX practice_turns_session_created_idx
    ON practice_turns (session_id, created_at ASC);
