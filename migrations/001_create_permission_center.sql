-- Permission center schema.
--
-- The application namespace is deliberately a string carried by roles and
-- resources. There is no separate application lifecycle in this service.
-- Deleted rows remain for audit; partial unique indexes scope business-key
-- uniqueness to live rows so a key can be recreated after a soft delete.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(80) PRIMARY KEY DEFAULT encode(gen_random_bytes(16), 'hex'),
    application VARCHAR(100) NOT NULL,
    code VARCHAR(128) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT roles_id_not_blank CHECK (btrim(id) <> ''),
    CONSTRAINT roles_application_not_blank CHECK (btrim(application) <> ''),
    CONSTRAINT roles_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT roles_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT roles_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'roles' AND column_name = 'deleted_at'
    ) THEN
        EXECUTE 'UPDATE roles SET is_deleted = TRUE WHERE deleted_at IS NOT NULL';
    END IF;
END;
$$;

ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_deleted_implies_disabled;
ALTER TABLE roles ADD CONSTRAINT roles_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled);

DROP INDEX IF EXISTS roles_application_code_live_uq;
CREATE UNIQUE INDEX IF NOT EXISTS roles_application_code_live_uq
    ON roles (application, code)
    WHERE is_deleted = FALSE;

DROP INDEX IF EXISTS roles_application_live_idx;
CREATE INDEX IF NOT EXISTS roles_application_live_idx
    ON roles (application, enabled, name)
    WHERE is_deleted = FALSE;

CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application VARCHAR(100) NOT NULL,
    parent_id UUID,
    resource_type VARCHAR(16) NOT NULL,
    code VARCHAR(160) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    route VARCHAR(500),
    component VARCHAR(500),
    icon VARCHAR(200),
    action VARCHAR(500),
    http_method VARCHAR(16),
    sort_order INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT resources_application_not_blank CHECK (btrim(application) <> ''),
    CONSTRAINT resources_parent_same_application_fk
        FOREIGN KEY (parent_id, application)
        REFERENCES resources (id, application) ON DELETE RESTRICT,
    CONSTRAINT resources_id_application_uq UNIQUE (id, application),
    CONSTRAINT resources_type_check CHECK (resource_type IN ('menu', 'button')),
    CONSTRAINT resources_parent_not_self_check CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT resources_button_requires_parent_check
        CHECK (resource_type <> 'button' OR parent_id IS NOT NULL),
    CONSTRAINT resources_button_api_path_check
        CHECK (resource_type <> 'button' OR btrim(action) <> ''),
    CONSTRAINT resources_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT resources_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT resources_sort_order_check CHECK (sort_order >= 0),
    CONSTRAINT resources_http_method_check
        CHECK (http_method IS NULL OR upper(http_method) IN
            ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS')),
    CONSTRAINT resources_metadata_object_check CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT resources_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

ALTER TABLE resources ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE resources ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE resources ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE resources ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE resources ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema() AND table_name = 'resources' AND column_name = 'deleted_at'
    ) THEN
        EXECUTE 'UPDATE resources SET is_deleted = TRUE WHERE deleted_at IS NOT NULL';
    END IF;
END;
$$;

ALTER TABLE resources DROP CONSTRAINT IF EXISTS resources_deleted_implies_disabled;
ALTER TABLE resources ADD CONSTRAINT resources_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled);

-- A code is the stable permission identifier. Keeping it unique across menu
-- and button nodes avoids ambiguous permission checks in callers.
DROP INDEX IF EXISTS resources_application_code_live_uq;
CREATE UNIQUE INDEX IF NOT EXISTS resources_application_code_live_uq
    ON resources (application, code)
    WHERE is_deleted = FALSE;

-- PostgreSQL treats NULL parent IDs as distinct in a normal unique constraint;
-- this expression index also prevents duplicate live root nodes with the same
-- type and name in one application namespace.
DROP INDEX IF EXISTS resources_sibling_name_live_uq;
CREATE UNIQUE INDEX IF NOT EXISTS resources_sibling_name_live_uq
    ON resources (application, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::UUID), resource_type, name)
    WHERE is_deleted = FALSE;

DROP INDEX IF EXISTS resources_application_tree_idx;
CREATE INDEX IF NOT EXISTS resources_application_tree_idx
    ON resources (application, parent_id, resource_type, sort_order, name)
    WHERE is_deleted = FALSE;

DROP INDEX IF EXISTS resources_enabled_idx;
CREATE INDEX IF NOT EXISTS resources_enabled_idx
    ON resources (application, enabled, resource_type, sort_order)
    WHERE is_deleted = FALSE;

CREATE TABLE IF NOT EXISTS role_resources (
    role_id VARCHAR(80) NOT NULL,
    resource_id UUID NOT NULL,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT role_resources_pk PRIMARY KEY (role_id, resource_id),
    CONSTRAINT role_resources_role_fk
        FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    CONSTRAINT role_resources_resource_fk
        FOREIGN KEY (resource_id) REFERENCES resources (id) ON DELETE CASCADE
);

ALTER TABLE role_resources ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE role_resources ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE role_resources ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE role_resources ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE role_resources ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS role_resources_resource_idx
    ON role_resources (resource_id, role_id);

CREATE TABLE IF NOT EXISTS user_roles (
    subject VARCHAR(80) NOT NULL,
    role_id VARCHAR(80) NOT NULL,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT user_roles_pk PRIMARY KEY (subject, role_id),
    CONSTRAINT user_roles_subject_not_blank CHECK (btrim(subject) <> ''),
    CONSTRAINT user_roles_role_id_not_blank CHECK (btrim(role_id) <> '')
);

ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system';
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system';
ALTER TABLE user_roles ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS user_roles_role_idx
    ON user_roles (role_id, subject);

CREATE OR REPLACE FUNCTION permission_set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS roles_set_updated_at ON roles;
CREATE TRIGGER roles_set_updated_at
BEFORE UPDATE ON roles
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS resources_set_updated_at ON resources;
CREATE TRIGGER resources_set_updated_at
BEFORE UPDATE ON resources
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS role_resources_set_updated_at ON role_resources;
CREATE TRIGGER role_resources_set_updated_at
BEFORE UPDATE ON role_resources
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS user_roles_set_updated_at ON user_roles;
CREATE TRIGGER user_roles_set_updated_at
BEFORE UPDATE ON user_roles
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

-- A resource tree is scoped to one application by the composite parent FK.
-- This trigger enforces the semantic part of the tree: every parent is a menu,
-- and changing a parent cannot introduce a cycle.
CREATE OR REPLACE FUNCTION permission_validate_resource_parent()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    parent_type VARCHAR(16);
    parent_is_deleted BOOLEAN;
BEGIN
    IF NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT resource_type, is_deleted
      INTO parent_type, parent_is_deleted
      FROM resources
     WHERE id = NEW.parent_id
       AND application = NEW.application;

    IF parent_type IS NULL THEN
        -- The FK will provide the canonical error if this is a new parent.
        RETURN NEW;
    END IF;

    IF parent_type <> 'menu' THEN
        RAISE EXCEPTION 'resource parent % must be a menu', NEW.parent_id
            USING ERRCODE = 'check_violation';
    END IF;

    IF parent_is_deleted THEN
        RAISE EXCEPTION 'resource parent % is deleted', NEW.parent_id
            USING ERRCODE = 'check_violation';
    END IF;

    IF EXISTS (
        WITH RECURSIVE ancestors(id) AS (
            SELECT NEW.parent_id
            UNION ALL
            SELECT r.parent_id
              FROM resources r
              JOIN ancestors a ON a.id = r.id
             WHERE r.application = NEW.application
               AND r.parent_id IS NOT NULL
        )
        SELECT 1 FROM ancestors WHERE id = NEW.id
    ) THEN
        RAISE EXCEPTION 'resource % cannot be its own ancestor', NEW.id
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS resources_validate_parent ON resources;
CREATE TRIGGER resources_validate_parent
BEFORE INSERT OR UPDATE OF parent_id, application, resource_type, is_deleted ON resources
FOR EACH ROW EXECUTE FUNCTION permission_validate_resource_parent();

-- Role and resource IDs are globally unique, but their application scope is
-- still checked here to prevent a cross-application grant.
CREATE OR REPLACE FUNCTION permission_validate_role_resource_application()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    role_application VARCHAR(100);
    resource_application VARCHAR(100);
BEGIN
    SELECT application INTO role_application FROM roles WHERE id = NEW.role_id;
    SELECT application INTO resource_application FROM resources WHERE id = NEW.resource_id;
    IF role_application IS DISTINCT FROM resource_application THEN
        RAISE EXCEPTION 'role % and resource % belong to different applications', NEW.role_id, NEW.resource_id
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS role_resources_validate_application ON role_resources;
CREATE TRIGGER role_resources_validate_application
BEFORE INSERT OR UPDATE OF role_id, resource_id ON role_resources
FOR EACH ROW EXECUTE FUNCTION permission_validate_role_resource_application();
