CREATE TABLE evaluation_reports (
    id uuid PRIMARY KEY,
    actor_id uuid NOT NULL,
    session_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('QUEUED', 'RUNNING', 'READY', 'FAILED')),
    schema_version text NOT NULL CHECK (schema_version = 'evaluation-report/v2'),
    version integer NOT NULL CHECK (version > 0),
    scene_type text NOT NULL CHECK (
        scene_type IN ('IELTS_SPEAKING', 'INTERVIEW', 'OVERSEAS_DAILY_LIFE', 'OVERSEAS_WORKPLACE')
    ),
    practice_experience text NOT NULL,
    scene_category text NOT NULL,
    practice_mode text NOT NULL,
    scoreability_status text CHECK (
        scoreability_status IN ('PROVISIONAL', 'INSUFFICIENT')
    ),
    summary text NOT NULL DEFAULT '',
    failure_code text,
    failure_message text,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (session_id, version),
    CHECK (
        (status = 'READY' AND scoreability_status IS NOT NULL AND completed_at IS NOT NULL
            AND failure_code IS NULL AND failure_message IS NULL)
        OR (status = 'FAILED' AND failure_code IS NOT NULL AND failure_message IS NOT NULL
            AND scoreability_status IS NULL)
        OR (status IN ('QUEUED', 'RUNNING') AND scoreability_status IS NULL
            AND completed_at IS NULL AND failure_code IS NULL AND failure_message IS NULL)
    )
);

CREATE INDEX evaluation_reports_actor_completed_idx
    ON evaluation_reports (actor_id, completed_at DESC, id DESC);
CREATE INDEX evaluation_reports_actor_session_idx
    ON evaluation_reports (actor_id, session_id, version DESC);

CREATE TABLE evaluation_report_questions (
    id uuid PRIMARY KEY,
    report_id uuid NOT NULL REFERENCES evaluation_reports(id) ON DELETE CASCADE,
    source_question_id uuid NOT NULL,
    parent_question_id uuid REFERENCES evaluation_report_questions(id) ON DELETE RESTRICT,
    position integer NOT NULL CHECK (position > 0),
    text text NOT NULL CHECK (btrim(text) <> ''),
    UNIQUE (report_id, position),
    UNIQUE (report_id, source_question_id)
);

CREATE TABLE evaluation_report_answers (
    id uuid PRIMARY KEY,
    report_question_id uuid NOT NULL UNIQUE
        REFERENCES evaluation_report_questions(id) ON DELETE CASCADE,
    source_turn_id uuid NOT NULL,
    transcript text NOT NULL CHECK (btrim(transcript) <> ''),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE evaluation_dimensions (
    id uuid PRIMARY KEY,
    report_id uuid NOT NULL REFERENCES evaluation_reports(id) ON DELETE CASCADE,
    dimension_key text NOT NULL CHECK (btrim(dimension_key) <> ''),
    score numeric(5,2),
    scale text NOT NULL CHECK (scale IN ('PERCENTAGE_100', 'IELTS_BAND_9')),
    coverage numeric(4,3) NOT NULL CHECK (coverage BETWEEN 0 AND 1),
    confidence numeric(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    reason_codes jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(reason_codes) = 'array'),
    position integer NOT NULL CHECK (position > 0),
    UNIQUE (report_id, dimension_key),
    UNIQUE (report_id, position),
    CHECK (
        score IS NULL
        OR (scale = 'PERCENTAGE_100' AND score BETWEEN 0 AND 100)
        OR (scale = 'IELTS_BAND_9' AND score BETWEEN 0 AND 9 AND mod(score * 2, 1) = 0)
    )
);
