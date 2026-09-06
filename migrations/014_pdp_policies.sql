-- Self-hosted PDP control plane. UI menus stay outside this authorization graph.
-- 005_add_pdp.sql used an incompatible application/action schema. Preserve it
-- under an explicit legacy name rather than silently reusing it as a PDP policy.
DO $$
BEGIN
    IF to_regclass('authorization_policies') IS NOT NULL
       AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'authorization_policies' AND column_name = 'application') THEN
        IF to_regclass('authorization_policy_bindings') IS NOT NULL THEN
            ALTER TABLE authorization_policy_bindings RENAME TO authorization_policy_bindings_legacy_014;
        END IF;
        ALTER TABLE authorization_policies RENAME TO authorization_policies_legacy_014;
    END IF;
END;
$$;

CREATE TABLE IF NOT EXISTS authorization_policies (
    id UUID PRIMARY KEY,
    service_resource VARCHAR(128) NOT NULL,
    code VARCHAR(160) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    effect VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    scope_level VARCHAR(16) NOT NULL DEFAULT 'api',
    priority INTEGER NOT NULL DEFAULT 0,
    condition JSONB,
    obligations JSONB NOT NULL DEFAULT '{}'::JSONB,
    current_version INTEGER NOT NULL DEFAULT 0,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT authorization_policies_service_resource_not_blank CHECK (btrim(service_resource) <> ''),
    CONSTRAINT authorization_policies_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT authorization_policies_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT authorization_policies_effect_check CHECK (effect IN ('allow', 'deny')),
    CONSTRAINT authorization_policies_status_check CHECK (status IN ('draft', 'published', 'disabled', 'archived')),
    CONSTRAINT authorization_policies_scope_check CHECK (scope_level IN ('api', 'row', 'field')),
    CONSTRAINT authorization_policies_condition_object_check CHECK (condition IS NULL OR jsonb_typeof(condition) = 'object'),
    CONSTRAINT authorization_policies_obligations_object_check CHECK (jsonb_typeof(obligations) = 'object'),
    CONSTRAINT authorization_policies_deleted_implies_archived CHECK (NOT is_deleted OR status = 'archived')
);
CREATE UNIQUE INDEX IF NOT EXISTS authorization_policies_service_resource_code_live_uq ON authorization_policies (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX IF NOT EXISTS authorization_policies_scope_status_live_idx ON authorization_policies (service_resource, status, priority DESC) WHERE is_deleted = FALSE;

CREATE TABLE IF NOT EXISTS authorization_policy_role_bindings (
    policy_id UUID NOT NULL REFERENCES authorization_policies (id) ON DELETE CASCADE,
    role_id VARCHAR(80) NOT NULL,
    PRIMARY KEY (policy_id, role_id)
);
CREATE INDEX IF NOT EXISTS authorization_policy_role_bindings_role_idx ON authorization_policy_role_bindings (role_id, policy_id);

CREATE TABLE IF NOT EXISTS authorization_policy_api_targets (
    policy_id UUID NOT NULL REFERENCES authorization_policies (id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES authorization_api_endpoints (id) ON DELETE RESTRICT,
    PRIMARY KEY (policy_id, endpoint_id)
);
CREATE INDEX IF NOT EXISTS authorization_policy_api_targets_endpoint_idx ON authorization_policy_api_targets (endpoint_id, policy_id);

CREATE TABLE IF NOT EXISTS authorization_policy_versions (
    policy_id UUID NOT NULL REFERENCES authorization_policies (id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    snapshot JSONB NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    published_by VARCHAR(80) NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (policy_id, version),
    CONSTRAINT authorization_policy_versions_version_check CHECK (version > 0),
    CONSTRAINT authorization_policy_versions_snapshot_object_check CHECK (jsonb_typeof(snapshot) = 'object')
);

CREATE TABLE IF NOT EXISTS authorization_decision_logs (
    decision_id UUID PRIMARY KEY,
    request_id VARCHAR(80) NOT NULL DEFAULT '',
    service_resource VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    subject_id VARCHAR(80) NOT NULL,
    endpoint_id UUID NOT NULL,
    decision VARCHAR(16) NOT NULL,
    reason_code VARCHAR(32) NOT NULL,
    matched_policy_ids UUID[] NOT NULL DEFAULT '{}',
    policy_snapshot_version INTEGER NOT NULL DEFAULT 0,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT authorization_decision_logs_decision_check CHECK (decision IN ('allow', 'deny'))
);
CREATE INDEX IF NOT EXISTS authorization_decision_logs_scope_endpoint_time_idx ON authorization_decision_logs (service_resource, endpoint_id, occurred_at DESC);

CREATE OR REPLACE FUNCTION permission_validate_policy_scope() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE policy_scope VARCHAR(128); bound_scope VARCHAR(128);
BEGIN
    SELECT service_resource INTO policy_scope FROM authorization_policies WHERE id = NEW.policy_id AND is_deleted = FALSE;
    IF TG_TABLE_NAME = 'authorization_policy_role_bindings' THEN
        SELECT service_resource INTO bound_scope FROM roles WHERE id = NEW.role_id AND is_deleted = FALSE;
    ELSE
        SELECT service_resource INTO bound_scope FROM authorization_api_endpoints WHERE id = NEW.endpoint_id AND is_deleted = FALSE;
    END IF;
    IF policy_scope IS NULL OR bound_scope IS NULL OR policy_scope <> bound_scope THEN
        RAISE EXCEPTION 'policy binding must remain in its service_resource' USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS authorization_policy_roles_validate_scope ON authorization_policy_role_bindings;
CREATE TRIGGER authorization_policy_roles_validate_scope BEFORE INSERT OR UPDATE ON authorization_policy_role_bindings FOR EACH ROW EXECUTE FUNCTION permission_validate_policy_scope();
DROP TRIGGER IF EXISTS authorization_policy_targets_validate_scope ON authorization_policy_api_targets;
CREATE TRIGGER authorization_policy_targets_validate_scope BEFORE INSERT OR UPDATE ON authorization_policy_api_targets FOR EACH ROW EXECUTE FUNCTION permission_validate_policy_scope();

DROP TRIGGER IF EXISTS authorization_policies_set_updated_at ON authorization_policies;
CREATE TRIGGER authorization_policies_set_updated_at BEFORE UPDATE ON authorization_policies FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

-- PDP is an administration control-plane page, so create its navigation entry
-- through the existing menu system. Visibility still comes from role_menus.
CREATE OR REPLACE FUNCTION permission_seed_pdp_navigation(p_service_resource VARCHAR)
RETURNS VOID LANGUAGE plpgsql AS $$
DECLARE permission_group_id UUID;
BEGIN
    SELECT id INTO permission_group_id FROM menus
     WHERE service_resource = p_service_resource AND code = 'permission-center' AND is_deleted = FALSE LIMIT 1;
    IF permission_group_id IS NULL THEN RETURN; END IF;
    INSERT INTO menus (service_resource,parent_id,menu_type,code,name,description,route,component,icon,sort_order,metadata,enabled)
    SELECT p_service_resource, permission_group_id, 'menu', 'policy-management', 'PDP 授权策略', '维护角色到 API 端点的 PDP 策略', '/policies', '/permission-center/policy-management/index.tsx', 'lock', 40, '{}'::JSONB, TRUE
    WHERE NOT EXISTS (SELECT 1 FROM menus WHERE service_resource=p_service_resource AND code='policy-management' AND is_deleted=FALSE);
END;
$$;
CREATE OR REPLACE FUNCTION permission_seed_pdp_navigation_after_resource_insert()
RETURNS TRIGGER LANGUAGE plpgsql AS $$ BEGIN PERFORM permission_seed_pdp_navigation(NEW.resource_key); RETURN NEW; END; $$;
DROP TRIGGER IF EXISTS service_resources_seed_pdp_navigation ON service_resources;
CREATE TRIGGER service_resources_seed_pdp_navigation AFTER INSERT ON service_resources FOR EACH ROW EXECUTE FUNCTION permission_seed_pdp_navigation_after_resource_insert();
SELECT permission_seed_pdp_navigation(resource_key) FROM service_resources WHERE is_deleted = FALSE;
