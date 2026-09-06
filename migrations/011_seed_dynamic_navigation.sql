-- 仪表盘由前端固定提供，其余侧栏和页面路由由 menus 表驱动。
-- 默认导航只在服务资源首次创建时写入，后续可通过菜单管理正常修改或删除。

CREATE OR REPLACE FUNCTION permission_seed_default_navigation(p_service_resource VARCHAR)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    permission_group_id UUID;
    basic_data_group_id UUID;
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
            p_service_resource, NULL, 'menu', 'permission-center', '权限中心', '角色与菜单权限管理',
            '/permission-center', NULL, 'lock', 10, '{}'::JSONB, TRUE
        ) RETURNING id INTO permission_group_id;
    END IF;

    INSERT INTO menus (
        service_resource, parent_id, menu_type, code, name, description,
        route, component, icon, sort_order, metadata, enabled
    )
    SELECT p_service_resource, permission_group_id, 'menu', definition.code, definition.name, definition.description,
           definition.route, definition.component, definition.icon, definition.sort_order, '{}'::JSONB, TRUE
      FROM (VALUES
        ('role-management', '角色管理', '维护服务资源下的角色和用户分配', '/roles', '/permission-center/role-management/index.tsx', 'usergroup', 10),
        ('menu-management', '菜单与按钮管理', '维护动态导航和按钮权限', '/menus', '/permission-center/menu-management/index.tsx', 'menu', 20)
      ) AS definition(code, name, description, route, component, icon, sort_order)
     WHERE NOT EXISTS (
        SELECT 1 FROM menus existing
         WHERE existing.service_resource = p_service_resource
           AND existing.code = definition.code
           AND existing.is_deleted = FALSE
    );

    SELECT id INTO basic_data_group_id
      FROM menus
     WHERE service_resource = p_service_resource
       AND code = 'basic-data'
       AND is_deleted = FALSE
     LIMIT 1;

    IF basic_data_group_id IS NULL THEN
        INSERT INTO menus (
            service_resource, parent_id, menu_type, code, name, description,
            route, component, icon, sort_order, metadata, enabled
        ) VALUES (
            p_service_resource, NULL, 'menu', 'basic-data', '基础数据', '权限中心基础目录',
            '/basic-data', NULL, 'cloud', 20, '{}'::JSONB, TRUE
        ) RETURNING id INTO basic_data_group_id;
    END IF;

    INSERT INTO menus (
        service_resource, parent_id, menu_type, code, name, description,
        route, component, icon, sort_order, metadata, enabled
    )
    SELECT p_service_resource, basic_data_group_id, 'menu', definition.code, definition.name, definition.description,
           definition.route, definition.component, definition.icon, definition.sort_order, '{}'::JSONB, TRUE
      FROM (VALUES
        ('service-resource-management', '服务资源管理', '维护服务资源目录', '/service-resources', '/permission-center/service-resource-management/index.tsx', 'cloud', 10),
        ('api-endpoint-management', 'API 端点管理', '维护和导入 API 端点', '/api-endpoints', '/permission-center/api-endpoint-management/index.tsx', 'api', 20)
      ) AS definition(code, name, description, route, component, icon, sort_order)
     WHERE NOT EXISTS (
        SELECT 1 FROM menus existing
         WHERE existing.service_resource = p_service_resource
           AND existing.code = definition.code
           AND existing.is_deleted = FALSE
    );
END;
$$;

CREATE OR REPLACE FUNCTION permission_seed_navigation_after_resource_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM permission_seed_default_navigation(NEW.resource_key);
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS service_resources_seed_default_navigation ON service_resources;
CREATE TRIGGER service_resources_seed_default_navigation
AFTER INSERT ON service_resources
FOR EACH ROW EXECUTE FUNCTION permission_seed_navigation_after_resource_insert();

SELECT permission_seed_default_navigation(resource_key)
  FROM service_resources
 WHERE is_deleted = FALSE;
