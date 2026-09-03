CREATE TABLE practice_answer_versions (
    id uuid PRIMARY KEY,
    actor_id uuid NOT NULL,
    source_question_id uuid NOT NULL,
    source_turn_id uuid NOT NULL,
    parent_version_id uuid REFERENCES practice_answer_versions(id) ON DELETE RESTRICT,
    version integer NOT NULL CHECK (version > 0),
    transcript text NOT NULL CHECK (btrim(transcript) <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_question_id, version),
    UNIQUE (source_turn_id),
    CHECK (parent_version_id IS NULL OR version > 1)
);

CREATE INDEX practice_answer_versions_actor_question_idx
    ON practice_answer_versions (actor_id, source_question_id, version DESC);

CREATE TABLE evaluation_retry_turns (
    id uuid PRIMARY KEY,
    actor_id uuid NOT NULL,
    feedback_item_id uuid NOT NULL
        REFERENCES evaluation_feedback_items(id) ON DELETE RESTRICT,
    report_id uuid NOT NULL REFERENCES evaluation_reports(id) ON DELETE RESTRICT,
    source_question_id uuid NOT NULL,
    answer_version_id uuid REFERENCES practice_answer_versions(id) ON DELETE SET NULL,
    repractice_mode text NOT NULL CHECK (repractice_mode = 'SAME_QUESTION'),
    idempotency_key text NOT NULL CHECK (
        btrim(idempotency_key) <> '' AND length(idempotency_key) <= 255
    ),
    status text NOT NULL CHECK (status IN ('PENDING', 'ACTIVE', 'COMPLETED', 'FAILED')),
    failure_code text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (actor_id, idempotency_key),
    UNIQUE (answer_version_id),
    CHECK (
        (status = 'FAILED' AND failure_code IS NOT NULL)
        OR (status <> 'FAILED' AND failure_code IS NULL)
    )
);

CREATE INDEX evaluation_retry_turns_feedback_idx
    ON evaluation_retry_turns (feedback_item_id, created_at DESC);
CREATE INDEX evaluation_retry_turns_report_idx
    ON evaluation_retry_turns (report_id, created_at DESC);
