CREATE TABLE IF NOT EXISTS applications (
    application VARCHAR(100) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT applications_application_not_blank CHECK (btrim(application) <> ''),
    CONSTRAINT applications_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT applications_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

INSERT INTO applications (application, name)
SELECT application, application FROM roles
ON CONFLICT (application) DO NOTHING;

INSERT INTO applications (application, name)
SELECT application, application FROM menus
ON CONFLICT (application) DO NOTHING;

CREATE INDEX IF NOT EXISTS applications_live_name_idx
    ON applications (name, application) WHERE is_deleted = FALSE;

CREATE OR REPLACE FUNCTION applications_set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$;

DROP TRIGGER IF EXISTS applications_set_updated_at ON applications;
CREATE TRIGGER applications_set_updated_at BEFORE UPDATE ON applications
FOR EACH ROW EXECUTE FUNCTION applications_set_updated_at();
