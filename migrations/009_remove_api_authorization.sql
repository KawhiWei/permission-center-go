-- 当前阶段只保留角色、菜单和角色菜单授权，移除 API 端点鉴权数据结构。
DROP TABLE IF EXISTS authorization_api_endpoint_roles;
DROP TABLE IF EXISTS authorization_api_endpoints;
DROP FUNCTION IF EXISTS permission_validate_api_endpoint_role_scope();
