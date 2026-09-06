-- Runnable PDP examples for permission.center.api. These records deliberately
-- cover unconditional allow, conditional allow, deny-overrides, and draft state.
DO $seed$
DECLARE
    service_value CONSTANT TEXT := 'permission.center.api';
    role_value TEXT;
    policy_value UUID;
    endpoint_value UUID;
    snapshot_text TEXT;
BEGIN
    SELECT id INTO role_value
    FROM roles
    WHERE service_resource = service_value
      AND enabled = TRUE
      AND is_deleted = FALSE
    ORDER BY created_at, id
    LIMIT 1;

    -- NexusAuth may provision the service before its first role is created.
    IF role_value IS NULL THEN
        RAISE NOTICE 'skip PDP examples: permission.center.api has no enabled role';
        RETURN;
    END IF;

    INSERT INTO user_roles (
        subject, role_id, created_by_id, created_by_name,
        updated_by_id, updated_by_name, is_deleted
    ) VALUES (
        'pdp-demo-user', role_value, 'pdp-example-seed', 'PDP 示例数据',
        'pdp-example-seed', 'PDP 示例数据', FALSE
    )
    ON CONFLICT (subject, role_id) DO UPDATE SET
        is_deleted = FALSE,
        updated_by_id = EXCLUDED.updated_by_id,
        updated_by_name = EXCLUDED.updated_by_name,
        updated_at = NOW();

    policy_value := '7d5e7c51-8a2e-4fa3-9001-000000000001';
    SELECT id INTO endpoint_value FROM authorization_api_endpoints
    WHERE service_resource = service_value AND method = 'GET'
      AND path_template = '/api/Clients' AND enabled = TRUE AND is_deleted = FALSE LIMIT 1;
    IF endpoint_value IS NOT NULL THEN
        INSERT INTO authorization_policies (
            id, service_resource, code, name, description, effect, status, scope_level, priority,
            condition, obligations, current_version, created_by_id, created_by_name, updated_by_id, updated_by_name
        ) VALUES (
            policy_value, service_value, 'example-api-read-allow', '示例：允许读取客户端',
            '无条件允许测试角色读取客户端列表，演示最基础的 API 级授权。', 'allow', 'draft', 'api', 100,
            NULL, '{}'::JSONB, 0, 'pdp-example-seed', 'PDP 示例数据', 'pdp-example-seed', 'PDP 示例数据'
        ) ON CONFLICT (id) DO NOTHING;
        INSERT INTO authorization_policy_role_bindings VALUES (policy_value, role_value) ON CONFLICT DO NOTHING;
        INSERT INTO authorization_policy_api_targets VALUES (policy_value, endpoint_value) ON CONFLICT DO NOTHING;
        snapshot_text := format('{"policy_id":"%s","version":1,"service_resource":"%s","effect":"allow","scope_level":"api","priority":100,"role_ids":["%s"],"endpoint_ids":["%s"],"obligations":{}}', policy_value, service_value, role_value, endpoint_value);
        INSERT INTO authorization_policy_versions (policy_id, version, snapshot, checksum, published_by)
        VALUES (policy_value, 1, snapshot_text::JSONB, encode(digest(snapshot_text, 'sha256'), 'hex'), 'pdp-example-seed')
        ON CONFLICT DO NOTHING;
        UPDATE authorization_policies SET status = 'published', current_version = 1 WHERE id = policy_value;
    END IF;

    policy_value := '7d5e7c51-8a2e-4fa3-9001-000000000002';
    SELECT id INTO endpoint_value FROM authorization_api_endpoints
    WHERE service_resource = service_value AND method = 'GET'
      AND path_template = '/api/users' AND enabled = TRUE AND is_deleted = FALSE LIMIT 1;
    IF endpoint_value IS NOT NULL THEN
        INSERT INTO authorization_policies (
            id, service_resource, code, name, description, effect, status, scope_level, priority,
            condition, obligations, current_version, created_by_id, created_by_name, updated_by_id, updated_by_name
        ) VALUES (
            policy_value, service_value, 'example-department-allow', '示例：工程部可读取用户',
            '仅当 subject.attributes.department 等于 engineering 时允许读取用户列表。', 'allow', 'draft', 'api', 90,
            '{"comparison":{"left":{"source":"subject","path":"department","type":"string"},"op":"eq","right":{"source":"literal","type":"string","value":"engineering"}}}'::JSONB,
            '{}'::JSONB, 0, 'pdp-example-seed', 'PDP 示例数据', 'pdp-example-seed', 'PDP 示例数据'
        ) ON CONFLICT (id) DO NOTHING;
        INSERT INTO authorization_policy_role_bindings VALUES (policy_value, role_value) ON CONFLICT DO NOTHING;
        INSERT INTO authorization_policy_api_targets VALUES (policy_value, endpoint_value) ON CONFLICT DO NOTHING;
        snapshot_text := format('{"policy_id":"%s","version":1,"service_resource":"%s","effect":"allow","scope_level":"api","priority":90,"role_ids":["%s"],"endpoint_ids":["%s"],"condition":{"comparison":{"left":{"source":"subject","path":"department","type":"string"},"op":"eq","right":{"source":"literal","type":"string","value":"engineering"}}},"obligations":{}}', policy_value, service_value, role_value, endpoint_value);
        INSERT INTO authorization_policy_versions (policy_id, version, snapshot, checksum, published_by)
        VALUES (policy_value, 1, snapshot_text::JSONB, encode(digest(snapshot_text, 'sha256'), 'hex'), 'pdp-example-seed')
        ON CONFLICT DO NOTHING;
        UPDATE authorization_policies SET status = 'published', current_version = 1 WHERE id = policy_value;
    END IF;

    SELECT id INTO endpoint_value FROM authorization_api_endpoints
    WHERE service_resource = service_value AND method = 'PATCH'
      AND path_template = '/api/users/{id}/status' AND enabled = TRUE AND is_deleted = FALSE LIMIT 1;
    IF endpoint_value IS NOT NULL THEN
        policy_value := '7d5e7c51-8a2e-4fa3-9001-000000000003';
        INSERT INTO authorization_policies (
            id, service_resource, code, name, description, effect, status, scope_level, priority,
            condition, obligations, current_version, created_by_id, created_by_name, updated_by_id, updated_by_name
        ) VALUES (
            policy_value, service_value, 'example-user-status-allow', '示例：允许修改用户状态',
            '允许测试角色修改用户状态；更高优先级的拒绝策略仍可覆盖本策略。', 'allow', 'draft', 'api', 100,
            NULL, '{}'::JSONB, 0, 'pdp-example-seed', 'PDP 示例数据', 'pdp-example-seed', 'PDP 示例数据'
        ) ON CONFLICT (id) DO NOTHING;
        INSERT INTO authorization_policy_role_bindings VALUES (policy_value, role_value) ON CONFLICT DO NOTHING;
        INSERT INTO authorization_policy_api_targets VALUES (policy_value, endpoint_value) ON CONFLICT DO NOTHING;
        snapshot_text := format('{"policy_id":"%s","version":1,"service_resource":"%s","effect":"allow","scope_level":"api","priority":100,"role_ids":["%s"],"endpoint_ids":["%s"],"obligations":{}}', policy_value, service_value, role_value, endpoint_value);
        INSERT INTO authorization_policy_versions (policy_id, version, snapshot, checksum, published_by)
        VALUES (policy_value, 1, snapshot_text::JSONB, encode(digest(snapshot_text, 'sha256'), 'hex'), 'pdp-example-seed')
        ON CONFLICT DO NOTHING;
        UPDATE authorization_policies SET status = 'published', current_version = 1 WHERE id = policy_value;

        policy_value := '7d5e7c51-8a2e-4fa3-9001-000000000004';
        INSERT INTO authorization_policies (
            id, service_resource, code, name, description, effect, status, scope_level, priority,
            condition, obligations, current_version, created_by_id, created_by_name, updated_by_id, updated_by_name
        ) VALUES (
            policy_value, service_value, 'example-suspended-deny', '示例：停用账号禁止改状态',
            '当 subject.attributes.account_status 等于 suspended 时显式拒绝，演示 deny-overrides。', 'deny', 'draft', 'api', 200,
            '{"comparison":{"left":{"source":"subject","path":"account_status","type":"string"},"op":"eq","right":{"source":"literal","type":"string","value":"suspended"}}}'::JSONB,
            '{}'::JSONB, 0, 'pdp-example-seed', 'PDP 示例数据', 'pdp-example-seed', 'PDP 示例数据'
        ) ON CONFLICT (id) DO NOTHING;
        INSERT INTO authorization_policy_role_bindings VALUES (policy_value, role_value) ON CONFLICT DO NOTHING;
        INSERT INTO authorization_policy_api_targets VALUES (policy_value, endpoint_value) ON CONFLICT DO NOTHING;
        snapshot_text := format('{"policy_id":"%s","version":1,"service_resource":"%s","effect":"deny","scope_level":"api","priority":200,"role_ids":["%s"],"endpoint_ids":["%s"],"condition":{"comparison":{"left":{"source":"subject","path":"account_status","type":"string"},"op":"eq","right":{"source":"literal","type":"string","value":"suspended"}}},"obligations":{}}', policy_value, service_value, role_value, endpoint_value);
        INSERT INTO authorization_policy_versions (policy_id, version, snapshot, checksum, published_by)
        VALUES (policy_value, 1, snapshot_text::JSONB, encode(digest(snapshot_text, 'sha256'), 'hex'), 'pdp-example-seed')
        ON CONFLICT DO NOTHING;
        UPDATE authorization_policies SET status = 'published', current_version = 1 WHERE id = policy_value;
    END IF;

    policy_value := '7d5e7c51-8a2e-4fa3-9001-000000000005';
    SELECT id INTO endpoint_value FROM authorization_api_endpoints
    WHERE service_resource = service_value AND method = 'POST'
      AND path_template = '/api/Clients' AND enabled = TRUE AND is_deleted = FALSE LIMIT 1;
    IF endpoint_value IS NOT NULL THEN
        INSERT INTO authorization_policies (
            id, service_resource, code, name, description, effect, status, scope_level, priority,
            condition, obligations, current_version, created_by_id, created_by_name, updated_by_id, updated_by_name
        ) VALUES (
            policy_value, service_value, 'example-client-create-draft', '示例：创建客户端（草稿）',
            '尚未发布的草稿策略不会参与 PDP 决策，可用于观察策略生命周期。', 'allow', 'draft', 'api', 80,
            '{"comparison":{"left":{"source":"subject","path":"department","type":"string"},"op":"eq","right":{"source":"literal","type":"string","value":"platform"}}}'::JSONB,
            '{}'::JSONB, 0, 'pdp-example-seed', 'PDP 示例数据', 'pdp-example-seed', 'PDP 示例数据'
        ) ON CONFLICT (id) DO NOTHING;
        INSERT INTO authorization_policy_role_bindings VALUES (policy_value, role_value) ON CONFLICT DO NOTHING;
        INSERT INTO authorization_policy_api_targets VALUES (policy_value, endpoint_value) ON CONFLICT DO NOTHING;
    END IF;
END
$seed$;
