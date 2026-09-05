-- 恢复统一服务资源目录，并为后续端点登记恢复最小数据结构。
-- 本迁移在 009_remove_api_authorization.sql 之后执行。

CREATE TABLE IF NOT EXISTS service_resources (
    resource_key VARCHAR(128) PRIMARY KEY,
    source VARCHAR(16) NOT NULL DEFAULT 'local',
    external_id VARCHAR(200),
    display_name VARCHAR(200) NOT NULL DEFAULT '',
    audience VARCHAR(200) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT service_resources_key_not_blank CHECK (btrim(resource_key) <> ''),
    CONSTRAINT service_resources_source_check CHECK (source IN ('local', 'nexusauth')),
    CONSTRAINT service_resources_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

-- Tolerate a partially created directory table while keeping this migration
-- safe to apply after an interrupted deployment.
ALTER TABLE service_resources
    ADD COLUMN IF NOT EXISTS source VARCHAR(16) NOT NULL DEFAULT 'local',
    ADD COLUMN IF NOT EXISTS external_id VARCHAR(200),
    ADD COLUMN IF NOT EXISTS display_name VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS audience VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
         WHERE conrelid = 'service_resources'::regclass
           AND conname = 'service_resources_key_not_blank'
    ) THEN
        ALTER TABLE service_resources
            ADD CONSTRAINT service_resources_key_not_blank CHECK (btrim(resource_key) <> '');
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
         WHERE conrelid = 'service_resources'::regclass
           AND conname = 'service_resources_source_check'
    ) THEN
        ALTER TABLE service_resources
            ADD CONSTRAINT service_resources_source_check CHECK (source IN ('local', 'nexusauth'));
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
         WHERE conrelid = 'service_resources'::regclass
           AND conname = 'service_resources_deleted_implies_disabled'
    ) THEN
        ALTER TABLE service_resources
            ADD CONSTRAINT service_resources_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled);
    END IF;
END;
$$;

-- Every historical permission scope remains available as a local directory
-- entry. ON CONFLICT preserves any row already synchronized or edited.
INSERT INTO service_resources (
    resource_key, source, display_name, audience, description, enabled,
    created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
)
SELECT resource_key, 'local', resource_key, resource_key, '', TRUE,
       'system', 'system', 'system', 'system', FALSE
  FROM (
      SELECT DISTINCT btrim(service_resource) AS resource_key
        FROM roles
       WHERE btrim(service_resource) <> ''
      UNION
      SELECT DISTINCT btrim(service_resource) AS resource_key
        FROM menus
       WHERE btrim(service_resource) <> ''
  ) AS historical_resources
ON CONFLICT (resource_key) DO NOTHING;

DROP TABLE IF EXISTS authorization_api_endpoint_roles;
DROP TABLE IF EXISTS authorization_api_endpoints CASCADE;

CREATE TABLE authorization_api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_resource VARCHAR(128) NOT NULL
        REFERENCES service_resources (resource_key) ON DELETE RESTRICT,
    controller VARCHAR(200) NOT NULL DEFAULT '',
    method VARCHAR(16) NOT NULL,
    path_template VARCHAR(500) NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT authorization_api_endpoints_service_resource_not_blank CHECK (btrim(service_resource) <> ''),
    CONSTRAINT authorization_api_endpoints_method_check CHECK (method = upper(method) AND btrim(method) <> ''),
    CONSTRAINT authorization_api_endpoints_path_not_blank CHECK (btrim(path_template) <> '' AND left(path_template, 1) = '/'),
    CONSTRAINT authorization_api_endpoints_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE UNIQUE INDEX authorization_api_endpoints_service_resource_route_live_uq
    ON authorization_api_endpoints (service_resource, method, path_template)
    WHERE is_deleted = FALSE;

CREATE INDEX authorization_api_endpoints_service_resource_live_idx
    ON authorization_api_endpoints (service_resource, enabled)
    WHERE is_deleted = FALSE;

DROP TRIGGER IF EXISTS service_resources_set_updated_at ON service_resources;
CREATE TRIGGER service_resources_set_updated_at
BEFORE UPDATE ON service_resources
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS authorization_api_endpoints_set_updated_at ON authorization_api_endpoints;
CREATE TRIGGER authorization_api_endpoints_set_updated_at
BEFORE UPDATE ON authorization_api_endpoints
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();
