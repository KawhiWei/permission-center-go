-- Move permission scope semantics from the legacy application alias to the
-- NexusAuth service-resource name. The legacy column remains populated for
-- old clients and is kept equal by a database trigger during the transition.

DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'roles', 'menus', 'authorization_resources', 'authorization_actions',
        'authorization_api_endpoints', 'authorization_policies'
    ] LOOP
        EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS service_resource VARCHAR(128) NOT NULL DEFAULT ''''', table_name);
        EXECUTE format('UPDATE %I SET service_resource = application WHERE btrim(service_resource) = ''''', table_name);
    END LOOP;
END;
$$;

-- PostgreSQL records UPDATE OF column lists as dependencies of a trigger. The
-- legacy menu and endpoint validators mention application explicitly, so they
-- must be removed while the compatibility column is widened below. They are
-- recreated after the canonical service-resource validators are installed.
DROP TRIGGER IF EXISTS menus_validate_parent ON menus;
DROP TRIGGER IF EXISTS authorization_api_endpoints_validate_scope ON authorization_api_endpoints;
DROP TRIGGER IF EXISTS roles_sync_service_resource ON roles;
DROP TRIGGER IF EXISTS menus_sync_service_resource ON menus;
DROP TRIGGER IF EXISTS authorization_resources_sync_service_resource ON authorization_resources;
DROP TRIGGER IF EXISTS authorization_actions_sync_service_resource ON authorization_actions;
DROP TRIGGER IF EXISTS authorization_api_endpoints_sync_service_resource ON authorization_api_endpoints;
DROP TRIGGER IF EXISTS authorization_policies_sync_service_resource ON authorization_policies;

-- NexusAuth API resource names are up to 128 characters. Widening the
-- compatibility alias avoids truncating a valid remote scope name.
ALTER TABLE roles ALTER COLUMN application TYPE VARCHAR(128);
ALTER TABLE menus ALTER COLUMN application TYPE VARCHAR(128);
ALTER TABLE applications ALTER COLUMN application TYPE VARCHAR(128);
ALTER TABLE authorization_resources ALTER COLUMN application TYPE VARCHAR(128);
ALTER TABLE authorization_actions ALTER COLUMN application TYPE VARCHAR(128);
ALTER TABLE authorization_api_endpoints ALTER COLUMN application TYPE VARCHAR(128);
ALTER TABLE authorization_policies ALTER COLUMN application TYPE VARCHAR(128);

CREATE OR REPLACE FUNCTION permission_sync_service_resource_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF btrim(NEW.service_resource) = '' THEN
        NEW.service_resource := NEW.application;
    ELSIF btrim(NEW.application) = '' THEN
        NEW.application := NEW.service_resource;
    ELSIF NEW.service_resource IS DISTINCT FROM NEW.application THEN
        RAISE EXCEPTION 'application and service_resource must identify the same scope'
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'roles', 'menus', 'authorization_resources', 'authorization_actions',
        'authorization_api_endpoints', 'authorization_policies'
    ] LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', table_name || '_sync_service_resource', table_name);
        EXECUTE format(
            'CREATE TRIGGER %I BEFORE INSERT OR UPDATE OF application, service_resource ON %I FOR EACH ROW EXECUTE FUNCTION permission_sync_service_resource_scope()',
            table_name || '_sync_service_resource', table_name
        );
    END LOOP;
END;
$$;

DO $$
DECLARE
    table_name TEXT;
    constraint_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'roles', 'menus', 'authorization_resources', 'authorization_actions',
        'authorization_api_endpoints', 'authorization_policies'
    ] LOOP
        constraint_name := table_name || '_service_resource_not_blank';
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conrelid = to_regclass(table_name) AND conname = constraint_name
        ) THEN
            EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK (btrim(service_resource) <> '''')', table_name, constraint_name);
        END IF;
    END LOOP;
END;
$$;

-- Keep service-resource scoped indexes independent from the legacy indexes so
-- new callers never have to rely on the compatibility column for lookups.
CREATE UNIQUE INDEX IF NOT EXISTS roles_service_resource_code_live_uq
    ON roles (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS roles_service_resource_live_idx
    ON roles (service_resource, enabled, name) WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS menus_service_resource_code_live_uq
    ON menus (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS menus_service_resource_tree_idx
    ON menus (service_resource, parent_id, menu_type, sort_order, name) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS menus_service_resource_enabled_idx
    ON menus (service_resource, enabled, menu_type, sort_order) WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_resources_service_resource_code_live_uq
    ON authorization_resources (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_resources_service_resource_live_idx
    ON authorization_resources (service_resource, resource_type, name) WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_actions_service_resource_code_live_uq
    ON authorization_actions (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_actions_service_resource_live_idx
    ON authorization_actions (service_resource, name) WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_api_endpoints_service_resource_route_live_uq
    ON authorization_api_endpoints (service_resource, service_code, method, path_template)
    WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_api_endpoints_service_resource_live_idx
    ON authorization_api_endpoints (service_resource, service_code, enabled)
    WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_policies_service_resource_code_live_uq
    ON authorization_policies (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_policies_service_resource_priority_live_idx
    ON authorization_policies (service_resource, priority DESC, id) WHERE is_deleted = FALSE;

-- Reapply the cross-table scope checks using the canonical service-resource
-- column. The legacy application column remains equal by the sync triggers.
CREATE OR REPLACE FUNCTION permission_validate_authorization_endpoint_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    resource_scope VARCHAR(128);
    action_scope VARCHAR(128);
BEGIN
    IF NEW.is_deleted THEN
        RETURN NEW;
    END IF;
    SELECT service_resource INTO resource_scope
      FROM authorization_resources WHERE id = NEW.resource_id AND is_deleted = FALSE;
    SELECT service_resource INTO action_scope
      FROM authorization_actions WHERE id = NEW.action_id AND is_deleted = FALSE;
    IF resource_scope IS DISTINCT FROM NEW.service_resource
       OR action_scope IS DISTINCT FROM NEW.service_resource THEN
        RAISE EXCEPTION 'endpoint, resource, and action must belong to the same service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION permission_validate_authorization_policy_binding_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    policy_scope VARCHAR(128);
    role_scope VARCHAR(128);
BEGIN
    IF NEW.is_deleted OR NEW.subject_type <> 'role' THEN
        RETURN NEW;
    END IF;
    SELECT service_resource INTO policy_scope
      FROM authorization_policies WHERE id = NEW.policy_id AND is_deleted = FALSE;
    SELECT service_resource INTO role_scope
      FROM roles WHERE id = NEW.subject_value AND is_deleted = FALSE;
    IF policy_scope IS DISTINCT FROM role_scope THEN
        RAISE EXCEPTION 'role and policy must belong to the same service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

-- Recreate validators removed before the compatibility-column type change.
-- Their event lists include service_resource so a canonical-scope update cannot
-- bypass validation. The menu validator reads the synchronized application
-- alias for compatibility with the existing composite parent key.
CREATE OR REPLACE FUNCTION permission_validate_menu_parent()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    parent_scope VARCHAR(128);
BEGIN
    IF NEW.is_deleted OR NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;
    SELECT service_resource INTO parent_scope
      FROM menus
     WHERE id = NEW.parent_id AND is_deleted = FALSE;
    IF parent_scope IS DISTINCT FROM NEW.service_resource THEN
        RAISE EXCEPTION 'menu parent and child must belong to the same service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER menus_validate_parent
BEFORE INSERT OR UPDATE OF parent_id, application, service_resource, menu_type, is_deleted ON menus
FOR EACH ROW EXECUTE FUNCTION permission_validate_menu_parent();

CREATE TRIGGER authorization_api_endpoints_validate_scope
BEFORE INSERT OR UPDATE OF application, service_resource, resource_id, action_id, is_deleted
ON authorization_api_endpoints
FOR EACH ROW EXECUTE FUNCTION permission_validate_authorization_endpoint_scope();
