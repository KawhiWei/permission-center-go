-- Upgrade an installation created by the original permission-center schema.
-- This migration is intentionally idempotent so it is also safe after the
-- audit-aware 001 migration has already been applied.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- These indexes and constraints reference deleted_at in the old schema. Drop
-- them before the column is removed, then recreate their live-row versions.
DROP INDEX IF EXISTS roles_application_code_live_uq;
DROP INDEX IF EXISTS roles_application_live_idx;
DROP INDEX IF EXISTS resources_application_code_live_uq;
DROP INDEX IF EXISTS resources_sibling_name_live_uq;
DROP INDEX IF EXISTS resources_application_tree_idx;
DROP INDEX IF EXISTS resources_enabled_idx;

ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_deleted_implies_disabled;
ALTER TABLE resources DROP CONSTRAINT IF EXISTS resources_deleted_implies_disabled;
DROP TRIGGER IF EXISTS resources_validate_parent ON resources;

ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE resources
    ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE role_resources
    ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE user_roles
    ADD COLUMN IF NOT EXISTS created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

-- Keep historical rows queryable and make their provenance explicit. Existing
-- non-empty audit values are preserved when this script is rerun.
UPDATE roles
SET created_by_id = CASE WHEN btrim(created_by_id) = '' THEN 'system' ELSE created_by_id END,
    created_by_name = CASE WHEN btrim(created_by_name) = '' THEN 'system' ELSE created_by_name END,
    updated_by_id = CASE WHEN btrim(updated_by_id) = '' THEN 'system' ELSE updated_by_id END,
    updated_by_name = CASE WHEN btrim(updated_by_name) = '' THEN 'system' ELSE updated_by_name END,
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

UPDATE resources
SET created_by_id = CASE WHEN btrim(created_by_id) = '' THEN 'system' ELSE created_by_id END,
    created_by_name = CASE WHEN btrim(created_by_name) = '' THEN 'system' ELSE created_by_name END,
    updated_by_id = CASE WHEN btrim(updated_by_id) = '' THEN 'system' ELSE updated_by_id END,
    updated_by_name = CASE WHEN btrim(updated_by_name) = '' THEN 'system' ELSE updated_by_name END,
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

UPDATE role_resources
SET created_by_id = CASE WHEN btrim(created_by_id) = '' THEN 'system' ELSE created_by_id END,
    created_by_name = CASE WHEN btrim(created_by_name) = '' THEN 'system' ELSE created_by_name END,
    updated_by_id = CASE WHEN btrim(updated_by_id) = '' THEN 'system' ELSE updated_by_id END,
    updated_by_name = CASE WHEN btrim(updated_by_name) = '' THEN 'system' ELSE updated_by_name END,
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

UPDATE user_roles
SET created_by_id = CASE WHEN btrim(created_by_id) = '' THEN 'system' ELSE created_by_id END,
    created_by_name = CASE WHEN btrim(created_by_name) = '' THEN 'system' ELSE created_by_name END,
    updated_by_id = CASE WHEN btrim(updated_by_id) = '' THEN 'system' ELSE updated_by_id END,
    updated_by_name = CASE WHEN btrim(updated_by_name) = '' THEN 'system' ELSE updated_by_name END,
    created_at = COALESCE(created_at, NOW()),
    updated_at = COALESCE(updated_at, NOW());

-- The old deleted_at value is the source of truth when upgrading. The OR
-- keeps an already-set is_deleted value intact on a partially upgraded run.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'roles' AND column_name = 'deleted_at'
    ) THEN
        UPDATE roles SET is_deleted = is_deleted OR deleted_at IS NOT NULL;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'resources' AND column_name = 'deleted_at'
    ) THEN
        UPDATE resources SET is_deleted = is_deleted OR deleted_at IS NOT NULL;
    END IF;
END
$$;

UPDATE roles SET enabled = FALSE WHERE is_deleted;
UPDATE resources SET enabled = FALSE WHERE is_deleted;

-- Some older deployments used UUID role identifiers. Converting through text
-- preserves every value while matching the string role-id contract.
ALTER TABLE roles DROP CONSTRAINT IF EXISTS role_resources_role_fk;
ALTER TABLE roles DROP CONSTRAINT IF EXISTS user_roles_role_fk;
ALTER TABLE role_resources DROP CONSTRAINT IF EXISTS role_resources_role_fk;
ALTER TABLE user_roles DROP CONSTRAINT IF EXISTS user_roles_role_fk;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'roles' AND column_name = 'id'
          AND udt_name = 'uuid'
    ) THEN
        ALTER TABLE roles ALTER COLUMN id DROP DEFAULT;
        ALTER TABLE roles ALTER COLUMN id TYPE VARCHAR(80) USING id::text;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'role_resources' AND column_name = 'role_id'
          AND udt_name = 'uuid'
    ) THEN
        ALTER TABLE role_resources ALTER COLUMN role_id TYPE VARCHAR(80) USING role_id::text;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'user_roles' AND column_name = 'role_id'
          AND udt_name = 'uuid'
    ) THEN
        ALTER TABLE user_roles ALTER COLUMN role_id TYPE VARCHAR(80) USING role_id::text;
    END IF;
END
$$;

ALTER TABLE roles ALTER COLUMN id SET DEFAULT encode(gen_random_bytes(16), 'hex');

ALTER TABLE roles
    ALTER COLUMN created_by_id SET DEFAULT 'system',
    ALTER COLUMN created_by_name SET DEFAULT 'system',
    ALTER COLUMN updated_by_id SET DEFAULT 'system',
    ALTER COLUMN updated_by_name SET DEFAULT 'system',
    ALTER COLUMN is_deleted SET DEFAULT FALSE,
    ALTER COLUMN created_by_id SET NOT NULL,
    ALTER COLUMN created_by_name SET NOT NULL,
    ALTER COLUMN updated_by_id SET NOT NULL,
    ALTER COLUMN updated_by_name SET NOT NULL,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN is_deleted SET NOT NULL;

ALTER TABLE resources
    ALTER COLUMN created_by_id SET DEFAULT 'system',
    ALTER COLUMN created_by_name SET DEFAULT 'system',
    ALTER COLUMN updated_by_id SET DEFAULT 'system',
    ALTER COLUMN updated_by_name SET DEFAULT 'system',
    ALTER COLUMN is_deleted SET DEFAULT FALSE,
    ALTER COLUMN created_by_id SET NOT NULL,
    ALTER COLUMN created_by_name SET NOT NULL,
    ALTER COLUMN updated_by_id SET NOT NULL,
    ALTER COLUMN updated_by_name SET NOT NULL,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN is_deleted SET NOT NULL;

ALTER TABLE role_resources
    ALTER COLUMN created_by_id SET DEFAULT 'system',
    ALTER COLUMN created_by_name SET DEFAULT 'system',
    ALTER COLUMN updated_by_id SET DEFAULT 'system',
    ALTER COLUMN updated_by_name SET DEFAULT 'system',
    ALTER COLUMN is_deleted SET DEFAULT FALSE,
    ALTER COLUMN created_by_id SET NOT NULL,
    ALTER COLUMN created_by_name SET NOT NULL,
    ALTER COLUMN updated_by_id SET NOT NULL,
    ALTER COLUMN updated_by_name SET NOT NULL,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN is_deleted SET NOT NULL;

ALTER TABLE user_roles
    ALTER COLUMN created_by_id SET DEFAULT 'system',
    ALTER COLUMN created_by_name SET DEFAULT 'system',
    ALTER COLUMN updated_by_id SET DEFAULT 'system',
    ALTER COLUMN updated_by_name SET DEFAULT 'system',
    ALTER COLUMN is_deleted SET DEFAULT FALSE,
    ALTER COLUMN created_by_id SET NOT NULL,
    ALTER COLUMN created_by_name SET NOT NULL,
    ALTER COLUMN updated_by_id SET NOT NULL,
    ALTER COLUMN updated_by_name SET NOT NULL,
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN updated_at SET NOT NULL,
    ALTER COLUMN is_deleted SET NOT NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'roles' AND column_name = 'deleted_at'
    ) THEN
        ALTER TABLE roles DROP COLUMN deleted_at;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'resources' AND column_name = 'deleted_at'
    ) THEN
        ALTER TABLE resources DROP COLUMN deleted_at;
    END IF;
END
$$;

ALTER TABLE roles
    ADD CONSTRAINT roles_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled);
ALTER TABLE resources
    ADD CONSTRAINT resources_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled);

ALTER TABLE role_resources
    ADD CONSTRAINT role_resources_role_fk
        FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;
ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_role_fk
        FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE;

CREATE UNIQUE INDEX IF NOT EXISTS roles_application_code_live_uq
    ON roles (application, code)
    WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS roles_application_live_idx
    ON roles (application, enabled, name)
    WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS resources_application_code_live_uq
    ON resources (application, code)
    WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS resources_sibling_name_live_uq
    ON resources (application, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::UUID), resource_type, name)
    WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS resources_application_tree_idx
    ON resources (application, parent_id, resource_type, sort_order, name)
    WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS resources_enabled_idx
    ON resources (application, enabled, resource_type, sort_order)
    WHERE is_deleted = FALSE;

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

CREATE TRIGGER resources_validate_parent
BEFORE INSERT OR UPDATE OF parent_id, application, resource_type, is_deleted ON resources
FOR EACH ROW EXECUTE FUNCTION permission_validate_resource_parent();
