CREATE TABLE practice_sessions (
    id uuid PRIMARY KEY,
    actor_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id uuid NOT NULL REFERENCES practice_plans(id) ON DELETE RESTRICT,
    scene_id text NOT NULL CHECK (btrim(scene_id) <> ''),
    scene_version integer NOT NULL CHECK (scene_version > 0),
    status text NOT NULL CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX practice_sessions_actor_created_idx ON practice_sessions (actor_id, created_at DESC);
CREATE INDEX practice_sessions_plan_idx ON practice_sessions (plan_id);

CREATE TABLE practice_questions (
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES practice_sessions(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    content text NOT NULL CHECK (btrim(content) <> ''),
    UNIQUE (session_id, position)
);

CREATE INDEX practice_questions_session_position_idx ON practice_questions (session_id, position ASC);
