-- 仪表盘是前端固定入口，不再作为服务资源的可配置菜单保存。
WITH backend_dashboards AS (
    SELECT id
      FROM menus
     WHERE code = 'dashboard'
       AND route = '/dashboard'
)
UPDATE menus
   SET parent_id = NULL,
       updated_by_id = 'system',
       updated_by_name = 'system',
       updated_at = NOW()
 WHERE parent_id IN (SELECT id FROM backend_dashboards);

DELETE FROM menus
 WHERE code = 'dashboard'
   AND route = '/dashboard';
