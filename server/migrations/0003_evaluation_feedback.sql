CREATE TABLE evaluation_evidence (
    id uuid PRIMARY KEY,
    report_id uuid NOT NULL REFERENCES evaluation_reports(id) ON DELETE CASCADE,
    source_turn_id uuid NOT NULL,
    start_utf8_byte integer NOT NULL CHECK (start_utf8_byte >= 0),
    end_utf8_byte integer NOT NULL CHECK (end_utf8_byte > start_utf8_byte),
    original_excerpt text NOT NULL CHECK (btrim(original_excerpt) <> ''),
    UNIQUE (report_id, id),
    UNIQUE (report_id, source_turn_id, start_utf8_byte, end_utf8_byte)
);

CREATE INDEX evaluation_evidence_report_turn_idx
    ON evaluation_evidence (report_id, source_turn_id);

CREATE TABLE evaluation_findings (
    id uuid PRIMARY KEY,
    report_id uuid NOT NULL REFERENCES evaluation_reports(id) ON DELETE CASCADE,
    dimension_id uuid NOT NULL REFERENCES evaluation_dimensions(id) ON DELETE CASCADE,
    finding_key text NOT NULL CHECK (btrim(finding_key) <> ''),
    kind text NOT NULL CHECK (kind IN ('STRENGTH', 'IMPROVEMENT', 'RECOMMENDED_EXAMPLE')),
    message text NOT NULL CHECK (btrim(message) <> ''),
    suggestion text,
    position integer NOT NULL CHECK (position > 0),
    UNIQUE (dimension_id, finding_key),
    UNIQUE (dimension_id, kind, position)
);

CREATE TABLE evaluation_finding_evidence (
    finding_id uuid NOT NULL REFERENCES evaluation_findings(id) ON DELETE CASCADE,
    evidence_id uuid NOT NULL REFERENCES evaluation_evidence(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position > 0),
    PRIMARY KEY (finding_id, evidence_id),
    UNIQUE (finding_id, position)
);

CREATE TABLE evaluation_feedback_items (
    id uuid PRIMARY KEY,
    report_id uuid NOT NULL REFERENCES evaluation_reports(id) ON DELETE CASCADE,
    finding_id uuid REFERENCES evaluation_findings(id) ON DELETE SET NULL,
    evidence_id uuid NOT NULL REFERENCES evaluation_evidence(id) ON DELETE RESTRICT,
    position integer NOT NULL CHECK (position > 0),
    category text NOT NULL CHECK (
        category IN ('CORRECTION', 'STRENGTH', 'RECOMMENDED_EXPRESSION')
    ),
    severity text CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH')),
    recommendation text NOT NULL CHECK (btrim(recommendation) <> ''),
    correction text,
    repractice_mode text NOT NULL CHECK (repractice_mode IN ('NONE', 'SAME_QUESTION')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (report_id, position),
    CHECK (
        (category = 'STRENGTH' AND correction IS NULL AND repractice_mode = 'NONE')
        OR (category IN ('CORRECTION', 'RECOMMENDED_EXPRESSION')
            AND correction IS NOT NULL AND btrim(correction) <> '')
    )
);

CREATE INDEX evaluation_feedback_report_idx
    ON evaluation_feedback_items (report_id, position);

CREATE TABLE evaluation_priority_actions (
    report_id uuid NOT NULL REFERENCES evaluation_reports(id) ON DELETE CASCADE,
    dimension_id uuid NOT NULL REFERENCES evaluation_dimensions(id) ON DELETE CASCADE,
    finding_id uuid NOT NULL REFERENCES evaluation_findings(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position BETWEEN 1 AND 5),
    PRIMARY KEY (report_id, finding_id),
    UNIQUE (report_id, position)
);
