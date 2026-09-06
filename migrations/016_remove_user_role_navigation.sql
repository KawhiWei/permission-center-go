-- User assignment now lives inside role management, so the standalone
-- user-role page is no longer part of the administration navigation.

UPDATE menus
SET enabled = FALSE,
    is_deleted = TRUE,
    updated_by_id = 'system',
    updated_by_name = 'system',
    updated_at = NOW()
WHERE code = 'user-role-management'
  AND is_deleted = FALSE;

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

    SELECT id INTO permission_group_id FROM menus
     WHERE service_resource = p_service_resource AND code = 'permission-center' AND is_deleted = FALSE LIMIT 1;
    IF permission_group_id IS NULL THEN
        INSERT INTO menus (service_resource,parent_id,menu_type,code,name,description,route,component,icon,sort_order,metadata,enabled)
        VALUES (p_service_resource,NULL,'menu','permission-center','权限中心','角色与授权策略管理','/permission-center',NULL,'lock',10,'{}'::JSONB,TRUE)
        RETURNING id INTO permission_group_id;
    END IF;

    INSERT INTO menus (service_resource,parent_id,menu_type,code,name,description,route,component,icon,sort_order,metadata,enabled)
    SELECT p_service_resource,permission_group_id,'menu',definition.code,definition.name,definition.description,
           definition.route,definition.component,definition.icon,definition.sort_order,'{}'::JSONB,TRUE
      FROM (VALUES
        ('role-management','角色管理','维护服务资源下的角色和用户分配','/roles','/permission-center/role-management/index.tsx','usergroup',10),
        ('menu-management','菜单与按钮管理','维护动态导航和按钮权限','/menus','/permission-center/menu-management/index.tsx','menu',20)
      ) AS definition(code,name,description,route,component,icon,sort_order)
     WHERE NOT EXISTS (
        SELECT 1 FROM menus existing
         WHERE existing.service_resource=p_service_resource AND existing.code=definition.code AND existing.is_deleted=FALSE
     );

    SELECT id INTO basic_data_group_id FROM menus
     WHERE service_resource=p_service_resource AND code='basic-data' AND is_deleted=FALSE LIMIT 1;
    IF basic_data_group_id IS NULL THEN
        INSERT INTO menus (service_resource,parent_id,menu_type,code,name,description,route,component,icon,sort_order,metadata,enabled)
        VALUES (p_service_resource,NULL,'menu','basic-data','基础数据','权限中心基础目录','/basic-data',NULL,'cloud',20,'{}'::JSONB,TRUE)
        RETURNING id INTO basic_data_group_id;
    END IF;

    INSERT INTO menus (service_resource,parent_id,menu_type,code,name,description,route,component,icon,sort_order,metadata,enabled)
    SELECT p_service_resource,basic_data_group_id,'menu',definition.code,definition.name,definition.description,
           definition.route,definition.component,definition.icon,definition.sort_order,'{}'::JSONB,TRUE
      FROM (VALUES
        ('service-resource-management','服务资源管理','维护服务资源目录','/service-resources','/permission-center/service-resource-management/index.tsx','cloud',10),
        ('api-endpoint-management','API 端点管理','维护和导入 API 端点','/api-endpoints','/permission-center/api-endpoint-management/index.tsx','api',20)
      ) AS definition(code,name,description,route,component,icon,sort_order)
     WHERE NOT EXISTS (
        SELECT 1 FROM menus existing
         WHERE existing.service_resource=p_service_resource AND existing.code=definition.code AND existing.is_deleted=FALSE
     );
END;
$$;
