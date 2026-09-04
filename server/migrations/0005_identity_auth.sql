CREATE TABLE users (
    id text PRIMARY KEY CHECK (btrim(id) <> ''),
    canonical_email text NOT NULL UNIQUE CHECK (
        canonical_email = lower(btrim(canonical_email))
        AND length(canonical_email) <= 254
    ),
    status text NOT NULL DEFAULT 'active' CHECK (status = 'active'),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE credentials (
    user_id text PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash text NOT NULL CHECK (btrim(password_hash) <> ''),
    updated_at timestamptz NOT NULL
);

CREATE TABLE auth_sessions (
    id text PRIMARY KEY CHECK (btrim(id) <> ''),
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_digest text NOT NULL UNIQUE CHECK (btrim(token_digest) <> ''),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (expires_at > created_at)
);

CREATE INDEX auth_sessions_user_id_idx ON auth_sessions (user_id);
CREATE INDEX auth_sessions_active_expiry_idx ON auth_sessions (expires_at) WHERE revoked_at IS NULL;
