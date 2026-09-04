CREATE TABLE practice_plans (
    id text PRIMARY KEY CHECK (btrim(id) <> ''),
    actor_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scene_id text NOT NULL CHECK (btrim(scene_id) <> ''),
    scene_version integer NOT NULL CHECK (scene_version > 0),
    role_id text NOT NULL CHECK (btrim(role_id) <> ''),
    practice_option_id text NOT NULL CHECK (btrim(practice_option_id) <> ''),
    objective text NOT NULL CHECK (btrim(objective) <> ''),
    status text NOT NULL CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX practice_plans_actor_created_idx ON practice_plans (actor_id, created_at DESC);
