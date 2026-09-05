-- 第一阶段只保留 API 端点级 RBAC：服务资源 -> API 端点 -> 角色 -> 用户。
-- 通用资源、动作和策略模型暂不保留，避免两套授权语义并行。

DROP TRIGGER IF EXISTS authorization_api_endpoints_validate_scope ON authorization_api_endpoints;
DROP TRIGGER IF EXISTS authorization_api_endpoints_sync_service_resource ON authorization_api_endpoints;
DROP TRIGGER IF EXISTS authorization_resources_sync_service_resource ON authorization_resources;
DROP TRIGGER IF EXISTS authorization_actions_sync_service_resource ON authorization_actions;
DROP TRIGGER IF EXISTS authorization_policies_sync_service_resource ON authorization_policies;
DROP TRIGGER IF EXISTS roles_sync_service_resource ON roles;
DROP TRIGGER IF EXISTS menus_sync_service_resource ON menus;
DROP TRIGGER IF EXISTS menus_validate_parent ON menus;

DROP INDEX IF EXISTS authorization_api_endpoints_route_live_uq;
DROP INDEX IF EXISTS authorization_api_endpoints_application_live_idx;
DROP INDEX IF EXISTS authorization_api_endpoints_service_resource_route_live_uq;

ALTER TABLE authorization_api_endpoints
    DROP CONSTRAINT IF EXISTS authorization_api_endpoints_application_not_blank,
    DROP COLUMN IF EXISTS application,
    DROP COLUMN IF EXISTS resource_id,
    DROP COLUMN IF EXISTS action_id,
    DROP COLUMN IF EXISTS enforcement_mode;

-- 同一服务资源内 Method + Path 就是一个端点，控制器仅用于分组展示。
WITH duplicate_endpoints AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY service_resource, method, path_template
               ORDER BY created_at, id
           ) AS row_number
      FROM authorization_api_endpoints
     WHERE is_deleted = FALSE
)
UPDATE authorization_api_endpoints endpoint
   SET is_deleted = TRUE, enabled = FALSE, updated_at = NOW()
  FROM duplicate_endpoints duplicate
 WHERE endpoint.id = duplicate.id AND duplicate.row_number > 1;

CREATE UNIQUE INDEX IF NOT EXISTS authorization_api_endpoints_service_resource_route_live_uq
    ON authorization_api_endpoints (service_resource, method, path_template)
    WHERE is_deleted = FALSE;

DROP TABLE IF EXISTS authorization_policy_bindings;
DROP TABLE IF EXISTS authorization_policies;
DROP TABLE IF EXISTS authorization_actions;
DROP TABLE IF EXISTS authorization_resources;

-- service_resource 已经是唯一业务作用域，删除角色和菜单中的兼容列。
-- 服务资源统一由 NexusAuth 提供，不再保留本地 applications 目录。
ALTER TABLE menus
    DROP CONSTRAINT IF EXISTS menus_parent_same_application_fk,
    DROP CONSTRAINT IF EXISTS menus_id_application_uq,
    DROP CONSTRAINT IF EXISTS menus_application_not_blank;
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_application_not_blank;

DROP INDEX IF EXISTS roles_application_code_live_uq;
DROP INDEX IF EXISTS roles_application_live_idx;
DROP INDEX IF EXISTS menus_application_code_live_uq;
DROP INDEX IF EXISTS menus_application_tree_idx;
DROP INDEX IF EXISTS menus_application_enabled_idx;

ALTER TABLE roles DROP COLUMN IF EXISTS application;
ALTER TABLE menus DROP COLUMN IF EXISTS application;
DROP TABLE IF EXISTS applications;

ALTER TABLE menus
    DROP CONSTRAINT IF EXISTS menus_parent_same_service_resource_fk,
    DROP CONSTRAINT IF EXISTS menus_id_service_resource_uq;
ALTER TABLE menus
    ADD CONSTRAINT menus_id_service_resource_uq UNIQUE (id, service_resource),
    ADD CONSTRAINT menus_parent_same_service_resource_fk
        FOREIGN KEY (parent_id, service_resource)
        REFERENCES menus (id, service_resource) ON DELETE RESTRICT;

CREATE OR REPLACE FUNCTION permission_validate_menu_parent()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    parent_type VARCHAR(16);
    parent_deleted BOOLEAN;
BEGIN
    IF NEW.is_deleted OR NEW.parent_id IS NULL THEN
        RETURN NEW;
    END IF;
    SELECT menu_type, is_deleted INTO parent_type, parent_deleted
      FROM menus
     WHERE id = NEW.parent_id AND service_resource = NEW.service_resource;
    IF parent_type IS DISTINCT FROM 'menu' OR parent_deleted THEN
        RAISE EXCEPTION 'menu parent must be an active menu in the same service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER menus_validate_parent
BEFORE INSERT OR UPDATE OF parent_id, service_resource, menu_type, is_deleted ON menus
FOR EACH ROW EXECUTE FUNCTION permission_validate_menu_parent();

CREATE TABLE IF NOT EXISTS authorization_api_endpoint_roles (
    endpoint_id UUID NOT NULL REFERENCES authorization_api_endpoints (id) ON DELETE CASCADE,
    role_id VARCHAR(80) NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT authorization_api_endpoint_roles_pk PRIMARY KEY (endpoint_id, role_id)
);

CREATE INDEX IF NOT EXISTS authorization_api_endpoint_roles_role_live_idx
    ON authorization_api_endpoint_roles (role_id, endpoint_id)
    WHERE is_deleted = FALSE;

CREATE OR REPLACE FUNCTION permission_validate_api_endpoint_role_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    endpoint_scope VARCHAR(128);
    role_scope VARCHAR(128);
BEGIN
    IF NEW.is_deleted THEN
        RETURN NEW;
    END IF;
    SELECT service_resource INTO endpoint_scope
      FROM authorization_api_endpoints
     WHERE id = NEW.endpoint_id AND is_deleted = FALSE;
    SELECT service_resource INTO role_scope
      FROM roles
     WHERE id = NEW.role_id AND is_deleted = FALSE;
    IF endpoint_scope IS NULL OR role_scope IS NULL OR endpoint_scope IS DISTINCT FROM role_scope THEN
        RAISE EXCEPTION 'endpoint and role must belong to the same service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS authorization_api_endpoint_roles_validate_scope
    ON authorization_api_endpoint_roles;
CREATE TRIGGER authorization_api_endpoint_roles_validate_scope
BEFORE INSERT OR UPDATE OF endpoint_id, role_id, is_deleted
ON authorization_api_endpoint_roles
FOR EACH ROW EXECUTE FUNCTION permission_validate_api_endpoint_role_scope();

DROP FUNCTION IF EXISTS permission_validate_authorization_endpoint_scope();
DROP FUNCTION IF EXISTS permission_validate_authorization_policy_binding_scope();
DROP FUNCTION IF EXISTS permission_sync_service_resource_scope();
