-- Phase A PDP configuration.
--
-- These tables describe authorization metadata only. Business instances and
-- their attributes stay in the owning service. All mutable rows use the same
-- audit fields and soft-delete convention as the existing RBAC tables.

CREATE TABLE IF NOT EXISTS authorization_resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application VARCHAR(100) NOT NULL,
    code VARCHAR(160) NOT NULL,
    resource_type VARCHAR(16) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    matcher VARCHAR(500) NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT authorization_resources_application_not_blank CHECK (btrim(application) <> ''),
    CONSTRAINT authorization_resources_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT authorization_resources_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT authorization_resources_type_check CHECK (resource_type IN ('api', 'entity')),
    CONSTRAINT authorization_resources_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE TABLE IF NOT EXISTS authorization_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application VARCHAR(100) NOT NULL,
    code VARCHAR(160) NOT NULL,
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
    CONSTRAINT authorization_actions_application_not_blank CHECK (btrim(application) <> ''),
    CONSTRAINT authorization_actions_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT authorization_actions_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT authorization_actions_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE TABLE IF NOT EXISTS authorization_api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application VARCHAR(100) NOT NULL,
    service_code VARCHAR(100) NOT NULL,
    method VARCHAR(16) NOT NULL,
    path_template VARCHAR(500) NOT NULL,
    resource_id UUID NOT NULL REFERENCES authorization_resources (id) ON DELETE RESTRICT,
    action_id UUID NOT NULL REFERENCES authorization_actions (id) ON DELETE RESTRICT,
    enforcement_mode VARCHAR(16) NOT NULL DEFAULT 'enforce',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT authorization_api_endpoints_application_not_blank CHECK (btrim(application) <> ''),
    CONSTRAINT authorization_api_endpoints_service_not_blank CHECK (btrim(service_code) <> ''),
    CONSTRAINT authorization_api_endpoints_method_check CHECK (method = upper(method) AND method IN
        ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS', 'TRACE')),
    CONSTRAINT authorization_api_endpoints_path_not_blank CHECK (btrim(path_template) <> '' AND left(path_template, 1) = '/'),
    CONSTRAINT authorization_api_endpoints_mode_check CHECK (enforcement_mode IN ('enforce', 'audit', 'disabled')),
    CONSTRAINT authorization_api_endpoints_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE TABLE IF NOT EXISTS authorization_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application VARCHAR(100) NOT NULL,
    code VARCHAR(160) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    effect VARCHAR(16) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    resource_codes JSONB NOT NULL DEFAULT '[]'::JSONB,
    action_codes JSONB NOT NULL DEFAULT '[]'::JSONB,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT authorization_policies_application_not_blank CHECK (btrim(application) <> ''),
    CONSTRAINT authorization_policies_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT authorization_policies_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT authorization_policies_effect_check CHECK (effect IN ('allow', 'deny')),
    CONSTRAINT authorization_policies_resource_codes_array_check CHECK (jsonb_typeof(resource_codes) = 'array'),
    CONSTRAINT authorization_policies_action_codes_array_check CHECK (jsonb_typeof(action_codes) = 'array'),
    CONSTRAINT authorization_policies_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE TABLE IF NOT EXISTS authorization_policy_bindings (
    policy_id UUID NOT NULL REFERENCES authorization_policies (id) ON DELETE CASCADE,
    subject_type VARCHAR(16) NOT NULL,
    subject_value VARCHAR(80) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT authorization_policy_bindings_pk PRIMARY KEY (policy_id, subject_type, subject_value),
    CONSTRAINT authorization_policy_bindings_subject_type_check CHECK (subject_type IN ('role', 'subject')),
    CONSTRAINT authorization_policy_bindings_subject_not_blank CHECK (btrim(subject_value) <> ''),
    CONSTRAINT authorization_policy_bindings_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE UNIQUE INDEX IF NOT EXISTS authorization_resources_application_code_live_uq
    ON authorization_resources (application, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_resources_application_live_idx
    ON authorization_resources (application, resource_type, name) WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_actions_application_code_live_uq
    ON authorization_actions (application, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_actions_application_live_idx
    ON authorization_actions (application, name) WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_api_endpoints_route_live_uq
    ON authorization_api_endpoints (application, service_code, method, path_template)
    WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_api_endpoints_application_live_idx
    ON authorization_api_endpoints (application, service_code, enabled)
    WHERE is_deleted = FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_policies_application_code_live_uq
    ON authorization_policies (application, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_policies_application_priority_live_idx
    ON authorization_policies (application, priority DESC, id) WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS authorization_policy_bindings_subject_live_idx
    ON authorization_policy_bindings (subject_type, subject_value, policy_id)
    WHERE is_deleted = FALSE AND enabled = TRUE;

DROP TRIGGER IF EXISTS authorization_resources_set_updated_at ON authorization_resources;
CREATE TRIGGER authorization_resources_set_updated_at
BEFORE UPDATE ON authorization_resources
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS authorization_actions_set_updated_at ON authorization_actions;
CREATE TRIGGER authorization_actions_set_updated_at
BEFORE UPDATE ON authorization_actions
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS authorization_api_endpoints_set_updated_at ON authorization_api_endpoints;
CREATE TRIGGER authorization_api_endpoints_set_updated_at
BEFORE UPDATE ON authorization_api_endpoints
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS authorization_policies_set_updated_at ON authorization_policies;
CREATE TRIGGER authorization_policies_set_updated_at
BEFORE UPDATE ON authorization_policies
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

DROP TRIGGER IF EXISTS authorization_policy_bindings_set_updated_at ON authorization_policy_bindings;
CREATE TRIGGER authorization_policy_bindings_set_updated_at
BEFORE UPDATE ON authorization_policy_bindings
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

-- The individual foreign keys above prevent missing references. These
-- triggers add the application-scope invariant that a plain UUID FK cannot
-- express.
CREATE OR REPLACE FUNCTION permission_validate_authorization_endpoint_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    resource_application VARCHAR(100);
    action_application VARCHAR(100);
BEGIN
    IF NEW.is_deleted THEN
        RETURN NEW;
    END IF;
    SELECT application INTO resource_application
      FROM authorization_resources WHERE id = NEW.resource_id AND is_deleted = FALSE;
    SELECT application INTO action_application
      FROM authorization_actions WHERE id = NEW.action_id AND is_deleted = FALSE;
    IF resource_application IS DISTINCT FROM NEW.application
       OR action_application IS DISTINCT FROM NEW.application THEN
        RAISE EXCEPTION 'endpoint, resource, and action must belong to the same application'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS authorization_api_endpoints_validate_scope ON authorization_api_endpoints;
CREATE TRIGGER authorization_api_endpoints_validate_scope
BEFORE INSERT OR UPDATE OF application, resource_id, action_id, is_deleted
ON authorization_api_endpoints
FOR EACH ROW EXECUTE FUNCTION permission_validate_authorization_endpoint_scope();

CREATE OR REPLACE FUNCTION permission_validate_authorization_policy_binding_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    policy_application VARCHAR(100);
    role_application VARCHAR(100);
BEGIN
    IF NEW.is_deleted THEN
        RETURN NEW;
    END IF;
    IF NEW.subject_type <> 'role' THEN
        RETURN NEW;
    END IF;
    SELECT application INTO policy_application
      FROM authorization_policies WHERE id = NEW.policy_id AND is_deleted = FALSE;
    SELECT application INTO role_application
      FROM roles WHERE id = NEW.subject_value AND is_deleted = FALSE;
    IF policy_application IS DISTINCT FROM role_application THEN
        RAISE EXCEPTION 'role and policy must belong to the same application'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS authorization_policy_bindings_validate_scope ON authorization_policy_bindings;
CREATE TRIGGER authorization_policy_bindings_validate_scope
BEFORE INSERT OR UPDATE OF policy_id, subject_type, subject_value, is_deleted
ON authorization_policy_bindings
FOR EACH ROW EXECUTE FUNCTION permission_validate_authorization_policy_binding_scope();
