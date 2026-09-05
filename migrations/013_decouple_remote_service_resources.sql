-- NexusAuth mode resolves service resources directly through the remote Open API.
-- Keep endpoint scopes as business keys and let the Go service validate them
-- against the configured source before writing endpoint records.
ALTER TABLE authorization_api_endpoints
    DROP CONSTRAINT IF EXISTS authorization_api_endpoints_service_resource_fkey;
