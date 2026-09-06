# Go 纯 HTTP 接入 Demo

该目录是一个业务服务示例，不使用 `sdk/pdp`。请求链路如下：

```text
客户端 Bearer JWT -> Demo 校验 OIDC JWT -> Demo 携带服务凭据调用权限中心 PDP -> allow 后执行业务接口
```

Demo 提供 `GET /api/orders/{id}`，并在 `GET /openapi.json` 暴露可导入权限中心的 OpenAPI 文档。

## 1. 配置权限中心

为权限中心和 Demo 生成同一个服务间密钥。密钥只放在两个服务的后端环境变量中，不能发给浏览器，也不能使用 OIDC Client Secret 代替。

```sh
export PERMISSION_CENTER_PDP_SERVICE_CREDENTIAL='change-to-a-long-random-secret'
docker compose up -d --build
```

在 NexusAuth 创建业务服务资源，例如 `orders-api`，其 audience 也建议使用 `orders-api`。确保登录用户取得的 access token 中包含：

- `sub`：NexusAuth 用户唯一标识。
- `aud`：包含 Demo 配置的 `orders-api`。
- `tenant_id`：当前租户。若实际 claim 名不同，可通过 `DEMO_TENANT_CLAIM` 修改。

## 2. 登记接口与策略

在权限中心 Dashboard 中：

1. 选择 `orders-api` 服务资源。
2. 在 API 端点页面手工创建 `GET /api/orders/{id}`，Controller 可填写 `Orders`。
3. 记录创建结果中该端点的 UUID。
4. 创建角色，并把 NexusAuth 用户的 `sub` 分配给该角色。
5. 创建 `allow`、`api` 级策略，绑定该角色和 API 端点，然后发布策略。

端点 UUID 由权限中心生成，必须配置在业务服务端，不能接受客户端通过 header、query 或 body 指定。
Demo 启动后也会在 `http://localhost:8090/openapi.json` 提供 OpenAPI 文档；接入更多路由时可以改用权限中心的 Swagger 导入功能批量登记。

## 3. 启动 Demo

```sh
cd demo/go
set -a
source .env.example
set +a
cd ../..
go run ./demo/go
```

请先把 `.env.example` 中的密钥、服务资源、端点 UUID、OIDC issuer 和 audience 替换为真实值。`.env.example` 仅是字段模板，不应写入真实密钥。

调用业务接口：

```sh
curl -i http://localhost:8090/api/orders/1001 \
  -H 'Authorization: Bearer <NexusAuth access token>' \
  -H 'X-Request-ID: demo-request-001'
```

预期结果：

- JWT 无效、过期、issuer/audience 不匹配，或缺少租户 claim：`401`。
- 权限中心返回 `deny`：`403`。
- 权限中心超时、不可用或服务凭据错误：`503`，请求不会进入业务处理。
- 权限中心返回 `allow`：`200` 并返回订单示例数据。

## PDP HTTP 请求

Demo 实际发送的请求如下。`subject.id` 和租户来自已验证 JWT；角色由权限中心根据 `subject.id + service_resource` 查询，调用方不得传入角色。

示例固定发送 `authorization_type: "api"`。`attributes` 是可选字段；配置的可信 claim 均不存在时，整个字段会从请求 JSON 中省略。

```http
POST /v1/pdp/decisions HTTP/1.1
Authorization: Bearer <DEMO_PDP_SERVICE_CREDENTIAL>
Content-Type: application/json

{
  "request_id": "demo-request-001",
  "authorization_type": "api",
  "service_resource": "orders-api",
  "tenant_id": "tenant-a",
  "subject": {
    "id": "nexusauth-user-sub",
    "attributes": {
      "department": "engineering"
    }
  },
  "action": {
    "kind": "http",
    "method": "GET"
  },
  "resource": {
    "kind": "api_endpoint",
    "endpoint_id": "<permission-center-endpoint-uuid>"
  }
}
```

生产环境应使用 HTTPS，并为 Demo 到权限中心的网络访问设置白名单或 mTLS。当前权限中心使用一个全局 PDP 服务凭据；若多个业务服务独立管理凭据，需要后续把服务凭据改造成按 `service_resource` 存储和校验。

## 真实 PDP 集成测试

`integration_test.go` 可以复用 Demo 的 HTTP PEP/PDP 代码调用一个正在运行的权限中心。测试身份由测试进程注入，仅用于隔离验证 PDP 链路；正式服务始终使用 `main.go` 中的 OIDC JWT 校验器。

```sh
DEMO_INTEGRATION=true \
DEMO_PDP_DECISION_URL=http://localhost:8080/v1/pdp/decisions \
DEMO_PDP_SERVICE_CREDENTIAL='<service credential>' \
DEMO_SERVICE_RESOURCE=permission.center.api \
DEMO_ENDPOINT_ID='<endpoint UUID>' \
DEMO_SUBJECT_ID='<NexusAuth subject>' \
DEMO_TENANT_ID=tenant-smoke \
go test ./demo/go -run TestPermissionCenterIntegration -v
```
