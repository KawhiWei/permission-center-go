-- 权限中心全量初始化脚本。
-- 项目从空数据库启动，不包含任何旧表升级、字段兼容或示例业务数据。

CREATE EXTENSION pgcrypto;

-- 服务资源是角色、菜单、API 端点和授权策略共享的业务作用域。
CREATE TABLE service_resources (
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

CREATE TABLE roles (
    id VARCHAR(80) PRIMARY KEY DEFAULT encode(gen_random_bytes(16), 'hex'),
    service_resource VARCHAR(128) NOT NULL,
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
    CONSTRAINT roles_service_resource_not_blank CHECK (btrim(service_resource) <> ''),
    CONSTRAINT roles_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT roles_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT roles_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE UNIQUE INDEX roles_service_resource_code_live_uq
    ON roles (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX roles_service_resource_live_idx
    ON roles (service_resource, enabled, name) WHERE is_deleted = FALSE;

-- menu 和 button 共享一棵树；按钮必须挂在菜单节点下并声明 API 路径。
CREATE TABLE menus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_resource VARCHAR(128) NOT NULL,
    parent_id UUID,
    menu_type VARCHAR(16) NOT NULL,
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
    CONSTRAINT menus_id_service_resource_uq UNIQUE (id, service_resource),
    CONSTRAINT menus_parent_same_service_resource_fk
        FOREIGN KEY (parent_id, service_resource)
        REFERENCES menus (id, service_resource) ON DELETE RESTRICT,
    CONSTRAINT menus_service_resource_not_blank CHECK (btrim(service_resource) <> ''),
    CONSTRAINT menus_type_check CHECK (menu_type IN ('menu', 'button')),
    CONSTRAINT menus_parent_not_self_check CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT menus_button_requires_parent_check CHECK (menu_type <> 'button' OR parent_id IS NOT NULL),
    CONSTRAINT menus_button_api_path_check CHECK (menu_type <> 'button' OR btrim(COALESCE(action, '')) <> ''),
    CONSTRAINT menus_code_not_blank CHECK (btrim(code) <> ''),
    CONSTRAINT menus_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT menus_sort_order_check CHECK (sort_order >= 0),
    CONSTRAINT menus_http_method_check CHECK (
        http_method IS NULL OR (http_method = upper(http_method) AND http_method IN
            ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS')
        )
    ),
    CONSTRAINT menus_metadata_object_check CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT menus_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE UNIQUE INDEX menus_service_resource_code_live_uq
    ON menus (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX menus_service_resource_tree_idx
    ON menus (service_resource, parent_id, menu_type, sort_order, name) WHERE is_deleted = FALSE;
CREATE INDEX menus_service_resource_enabled_idx
    ON menus (service_resource, enabled, menu_type, sort_order) WHERE is_deleted = FALSE;

CREATE TABLE role_menus (
    role_id VARCHAR(80) NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    menu_id UUID NOT NULL REFERENCES menus (id) ON DELETE CASCADE,
    created_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    created_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by_id VARCHAR(80) NOT NULL DEFAULT 'system',
    updated_by_name VARCHAR(200) NOT NULL DEFAULT 'system',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT role_menus_pk PRIMARY KEY (role_id, menu_id)
);

CREATE INDEX role_menus_menu_idx ON role_menus (menu_id, role_id);

CREATE TABLE user_roles (
    subject VARCHAR(80) NOT NULL,
    role_id VARCHAR(80) NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
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

CREATE INDEX user_roles_role_idx ON user_roles (role_id, subject);

-- API 端点使用服务资源业务键划分作用域。服务资源可由远程 NexusAuth 提供，
-- 因此这里不建立到本地 service_resources 目录的外键。
CREATE TABLE authorization_api_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_resource VARCHAR(128) NOT NULL,
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
    CONSTRAINT authorization_api_endpoints_path_not_blank CHECK (
        btrim(path_template) <> '' AND left(path_template, 1) = '/'
    ),
    CONSTRAINT authorization_api_endpoints_deleted_implies_disabled CHECK (NOT is_deleted OR NOT enabled)
);

CREATE UNIQUE INDEX authorization_api_endpoints_service_resource_route_live_uq
    ON authorization_api_endpoints (service_resource, method, path_template) WHERE is_deleted = FALSE;
CREATE INDEX authorization_api_endpoints_service_resource_live_idx
    ON authorization_api_endpoints (service_resource, enabled) WHERE is_deleted = FALSE;

-- authorization_type 明确区分 API 接口授权和携带业务对象属性的数据授权。
-- attributes 只存在于请求和策略条件中，不需要作为策略表固定列。
CREATE TABLE authorization_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_resource VARCHAR(128) NOT NULL,
    code VARCHAR(160) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    effect VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    authorization_type VARCHAR(16) NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    condition JSONB,
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
    CONSTRAINT authorization_policies_authorization_type_check CHECK (authorization_type IN ('api', 'data')),
    CONSTRAINT authorization_policies_condition_object_check CHECK (
        condition IS NULL OR jsonb_typeof(condition) = 'object'
    ),
    CONSTRAINT authorization_policies_current_version_check CHECK (current_version >= 0),
    CONSTRAINT authorization_policies_deleted_implies_archived CHECK (NOT is_deleted OR status = 'archived')
);

CREATE UNIQUE INDEX authorization_policies_service_resource_code_live_uq
    ON authorization_policies (service_resource, code) WHERE is_deleted = FALSE;
CREATE INDEX authorization_policies_scope_status_live_idx
    ON authorization_policies (service_resource, authorization_type, status, priority DESC)
    WHERE is_deleted = FALSE;

CREATE TABLE authorization_policy_role_bindings (
    policy_id UUID NOT NULL REFERENCES authorization_policies (id) ON DELETE CASCADE,
    role_id VARCHAR(80) NOT NULL,
    CONSTRAINT authorization_policy_role_bindings_pk PRIMARY KEY (policy_id, role_id),
    CONSTRAINT authorization_policy_role_bindings_role_not_blank CHECK (btrim(role_id) <> '')
);

CREATE INDEX authorization_policy_role_bindings_role_idx
    ON authorization_policy_role_bindings (role_id, policy_id);

CREATE TABLE authorization_policy_api_targets (
    policy_id UUID NOT NULL REFERENCES authorization_policies (id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES authorization_api_endpoints (id) ON DELETE RESTRICT,
    CONSTRAINT authorization_policy_api_targets_pk PRIMARY KEY (policy_id, endpoint_id)
);

CREATE INDEX authorization_policy_api_targets_endpoint_idx
    ON authorization_policy_api_targets (endpoint_id, policy_id);

-- 每次发布保存不可变快照；PDP 只读取当前发布版本对应的快照。
CREATE TABLE authorization_policy_versions (
    policy_id UUID NOT NULL REFERENCES authorization_policies (id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    snapshot JSONB NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    published_by VARCHAR(80) NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT authorization_policy_versions_pk PRIMARY KEY (policy_id, version),
    CONSTRAINT authorization_policy_versions_version_check CHECK (version > 0),
    CONSTRAINT authorization_policy_versions_snapshot_object_check CHECK (jsonb_typeof(snapshot) = 'object'),
    CONSTRAINT authorization_policy_versions_checksum_not_blank CHECK (btrim(checksum) <> ''),
    CONSTRAINT authorization_policy_versions_published_by_not_blank CHECK (btrim(published_by) <> '')
);

-- 决策日志不关联策略和端点外键，确保业务配置删除后审计记录仍可保留。
CREATE TABLE authorization_decision_logs (
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
    CONSTRAINT authorization_decision_logs_service_resource_not_blank CHECK (btrim(service_resource) <> ''),
    CONSTRAINT authorization_decision_logs_tenant_not_blank CHECK (btrim(tenant_id) <> ''),
    CONSTRAINT authorization_decision_logs_subject_not_blank CHECK (btrim(subject_id) <> ''),
    CONSTRAINT authorization_decision_logs_decision_check CHECK (decision IN ('allow', 'deny')),
    CONSTRAINT authorization_decision_logs_reason_not_blank CHECK (btrim(reason_code) <> ''),
    CONSTRAINT authorization_decision_logs_snapshot_version_check CHECK (policy_snapshot_version >= 0),
    CONSTRAINT authorization_decision_logs_latency_check CHECK (latency_ms >= 0)
);

CREATE INDEX authorization_decision_logs_scope_endpoint_time_idx
    ON authorization_decision_logs (service_resource, endpoint_id, occurred_at DESC);

-- 统一维护所有带审计字段表的 updated_at。
CREATE FUNCTION permission_set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER service_resources_set_updated_at
BEFORE UPDATE ON service_resources
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

CREATE TRIGGER roles_set_updated_at
BEFORE UPDATE ON roles
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

CREATE TRIGGER menus_set_updated_at
BEFORE UPDATE ON menus
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

CREATE TRIGGER role_menus_set_updated_at
BEFORE UPDATE ON role_menus
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

CREATE TRIGGER user_roles_set_updated_at
BEFORE UPDATE ON user_roles
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

CREATE TRIGGER authorization_api_endpoints_set_updated_at
BEFORE UPDATE ON authorization_api_endpoints
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

CREATE TRIGGER authorization_policies_set_updated_at
BEFORE UPDATE ON authorization_policies
FOR EACH ROW EXECUTE FUNCTION permission_set_updated_at();

-- 校验菜单父节点有效、同服务资源且类型为 menu，并阻止形成循环树。
CREATE FUNCTION permission_validate_menu_parent()
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

    SELECT menu_type, is_deleted
      INTO parent_type, parent_deleted
      FROM menus
     WHERE id = NEW.parent_id
       AND service_resource = NEW.service_resource;

    IF parent_type IS NULL OR parent_type <> 'menu' OR parent_deleted THEN
        RAISE EXCEPTION 'menu parent must be an active menu in the same service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;

    IF EXISTS (
        WITH RECURSIVE ancestors(id, parent_id) AS (
            SELECT id, parent_id FROM menus WHERE id = NEW.parent_id
            UNION ALL
            SELECT m.id, m.parent_id
              FROM menus m
              JOIN ancestors a ON m.id = a.parent_id
             WHERE m.service_resource = NEW.service_resource
        )
        SELECT 1 FROM ancestors WHERE id = NEW.id
    ) THEN
        RAISE EXCEPTION 'menu cannot be its own ancestor'
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER menus_validate_parent
BEFORE INSERT OR UPDATE OF parent_id, service_resource, menu_type, is_deleted ON menus
FOR EACH ROW EXECUTE FUNCTION permission_validate_menu_parent();

-- 防止角色获得其他服务资源的菜单或按钮。
CREATE FUNCTION permission_validate_role_menu_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    role_scope VARCHAR(128);
    menu_scope VARCHAR(128);
BEGIN
    IF NEW.is_deleted THEN
        RETURN NEW;
    END IF;

    SELECT service_resource INTO role_scope
      FROM roles WHERE id = NEW.role_id AND is_deleted = FALSE;
    SELECT service_resource INTO menu_scope
      FROM menus WHERE id = NEW.menu_id AND is_deleted = FALSE;

    IF role_scope IS NULL OR menu_scope IS NULL OR role_scope <> menu_scope THEN
        RAISE EXCEPTION 'role and menu must belong to the same service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER role_menus_validate_scope
BEFORE INSERT OR UPDATE OF role_id, menu_id, is_deleted ON role_menus
FOR EACH ROW EXECUTE FUNCTION permission_validate_role_menu_scope();

-- 策略只能绑定同一服务资源内未删除的角色和 API 端点。
CREATE FUNCTION permission_validate_policy_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    policy_scope VARCHAR(128);
    bound_scope VARCHAR(128);
BEGIN
    SELECT service_resource INTO policy_scope
      FROM authorization_policies
     WHERE id = NEW.policy_id AND is_deleted = FALSE;

    IF TG_TABLE_NAME = 'authorization_policy_role_bindings' THEN
        SELECT service_resource INTO bound_scope
          FROM roles WHERE id = NEW.role_id AND is_deleted = FALSE;
    ELSE
        SELECT service_resource INTO bound_scope
          FROM authorization_api_endpoints WHERE id = NEW.endpoint_id AND is_deleted = FALSE;
    END IF;

    IF policy_scope IS NULL OR bound_scope IS NULL OR policy_scope <> bound_scope THEN
        RAISE EXCEPTION 'policy binding must remain in its service resource'
            USING ERRCODE = 'foreign_key_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER authorization_policy_roles_validate_scope
BEFORE INSERT OR UPDATE ON authorization_policy_role_bindings
FOR EACH ROW EXECUTE FUNCTION permission_validate_policy_scope();

CREATE TRIGGER authorization_policy_targets_validate_scope
BEFORE INSERT OR UPDATE ON authorization_policy_api_targets
FOR EACH ROW EXECUTE FUNCTION permission_validate_policy_scope();

-- 创建服务资源时生成后台的基础导航。后台首页由前端固定提供。
CREATE FUNCTION permission_seed_default_navigation(p_service_resource VARCHAR)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    permission_group_id UUID;
    developer_group_id UUID;
BEGIN
    IF btrim(COALESCE(p_service_resource, '')) = '' THEN
        RETURN;
    END IF;

    SELECT id INTO permission_group_id
      FROM menus
     WHERE service_resource = p_service_resource
       AND code = 'permission-center'
       AND is_deleted = FALSE
     LIMIT 1;

    IF permission_group_id IS NULL THEN
        INSERT INTO menus (
            service_resource, parent_id, menu_type, code, name, description,
            route, component, icon, sort_order, metadata, enabled
        ) VALUES (
            p_service_resource, NULL, 'menu', 'permission-center', '权限中心',
            '角色与授权策略管理', '/permission-center', NULL, 'lock', 10, '{}'::JSONB, TRUE
        ) RETURNING id INTO permission_group_id;
    END IF;

    INSERT INTO menus (
        service_resource, parent_id, menu_type, code, name, description,
        route, component, icon, sort_order, metadata, enabled
    )
    SELECT p_service_resource, permission_group_id, 'menu', definition.code,
           definition.name, definition.description, definition.route,
           definition.component, definition.icon, definition.sort_order, '{}'::JSONB, TRUE
      FROM (VALUES
        ('role-management', '角色管理', '维护服务资源下的角色和用户分配',
         '/roles', '/permission-center/role-management/index.tsx', 'usergroup', 10),
        ('policy-management', 'PDP 授权策略', '维护 API 和数据级 PDP 授权策略',
         '/policies', '/permission-center/policy-management/index.tsx', 'lock', 40)
      ) AS definition(code, name, description, route, component, icon, sort_order)
     WHERE NOT EXISTS (
        SELECT 1 FROM menus existing
         WHERE existing.service_resource = p_service_resource
           AND existing.code = definition.code
           AND existing.is_deleted = FALSE
     );

    -- 服务资源、API 端点和菜单配置统一放在“开发者中心”目录下。
    SELECT id INTO developer_group_id
      FROM menus
     WHERE service_resource = p_service_resource
       AND code = 'developer-center'
       AND is_deleted = FALSE
     LIMIT 1;

    IF developer_group_id IS NULL THEN
        INSERT INTO menus (
            service_resource, parent_id, menu_type, code, name, description,
            route, component, icon, sort_order, metadata, enabled
        ) VALUES (
            p_service_resource, NULL, 'menu', 'developer-center', '开发者中心',
            '开发者工具与系统配置', '/developer-center', NULL, 'tools', 30, '{}'::JSONB, TRUE
        ) RETURNING id INTO developer_group_id;
    END IF;

    INSERT INTO menus (
        service_resource, parent_id, menu_type, code, name, description,
        route, component, icon, sort_order, metadata, enabled
    )
    SELECT p_service_resource, developer_group_id, 'menu', definition.code,
           definition.name, definition.description, definition.route,
           definition.component, definition.icon, definition.sort_order, '{}'::JSONB, TRUE
      FROM (VALUES
        ('service-resource-management', '服务资源管理', '维护服务资源目录',
         '/service-resources', '/permission-center/service-resource-management/index.tsx', 'cloud', 10),
        ('api-endpoint-management', 'API 端点管理', '维护和导入 API 端点',
         '/api-endpoints', '/permission-center/api-endpoint-management/index.tsx', 'api', 20),
        ('menu-management', '菜单管理', '维护动态导航和按钮权限',
         '/menus', '/permission-center/menu-management/index.tsx', 'menu', 30)
      ) AS definition(code, name, description, route, component, icon, sort_order)
     WHERE NOT EXISTS (
        SELECT 1 FROM menus existing
         WHERE existing.service_resource = p_service_resource
           AND existing.code = definition.code
           AND existing.is_deleted = FALSE
     );
END;
$$;

CREATE FUNCTION permission_seed_default_navigation_after_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM permission_seed_default_navigation(NEW.resource_key);
    RETURN NEW;
END;
$$;

CREATE TRIGGER service_resources_seed_default_navigation
AFTER INSERT ON service_resources
FOR EACH ROW EXECUTE FUNCTION permission_seed_default_navigation_after_insert();
