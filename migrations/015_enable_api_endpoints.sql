-- API endpoints imported from Swagger/OpenAPI are immediately available to
-- PDP policy configuration. Preserve soft-deleted rows as disabled.
UPDATE authorization_api_endpoints
   SET enabled = TRUE,
       updated_by_id = 'system',
       updated_by_name = 'migration-015',
       updated_at = NOW()
 WHERE is_deleted = FALSE
   AND enabled = FALSE;
