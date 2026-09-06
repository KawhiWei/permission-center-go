# .NET 纯 HTTP 接入 Demo

这是一个 ASP.NET Core 8 业务服务示例。它使用标准 JWT Bearer 中间件验证 NexusAuth access token，再通过 `HttpClient` 直接调用权限中心 PDP，不依赖权限中心 SDK。

## 权限中心准备

1. 权限中心配置 `PERMISSION_CENTER_PDP_SERVICE_CREDENTIAL`，Demo 的 `DEMO_PDP_SERVICE_CREDENTIAL` 必须使用相同值。
2. 创建或选择 `orders-api` 服务资源。
3. 登记 `GET /api/orders/{id}`，记录权限中心返回的 endpoint UUID。
4. 创建角色，把 NexusAuth 用户 `sub` 分配给该角色。
5. 创建并发布绑定该角色和 endpoint 的 `allow`、`api` 级策略。

JWT 必须包含 `sub`、`tenant_id`，且 issuer 和 audience 与配置一致。租户 claim 名可以通过 `DEMO_TENANT_CLAIM` 修改。

## 启动

编辑 `.env.example` 中的占位值，然后从仓库根目录运行：

```sh
set -a
source demo/dotnet/.env.example
set +a
dotnet run --project demo/dotnet/PermissionCenter.HttpDemo.csproj
```

本地示例允许使用 HTTP OIDC metadata；生产环境删除 `DEMO_OIDC_ALLOW_HTTP=true` 并使用 HTTPS。

## 调用

```sh
curl -i http://localhost:8091/api/orders/1001 \
  -H 'Authorization: Bearer <NexusAuth access token>' \
  -H 'X-Request-ID: dotnet-demo-001'
```

- JWT 验证失败或缺少身份/租户 claim：`401`。
- PDP 返回 `deny`：`403`。
- PDP 超时、不可用、响应无效或服务凭据错误：`503`，业务代码不会执行。
- PDP 返回 `allow`：`200`。

Demo 还提供公开的 `GET /healthz` 和 `GET /openapi.json`。后者可供权限中心 Swagger 导入功能使用。

PDP 请求体中的 `subject.id`、`tenant_id` 和属性来自已验证 JWT；endpoint UUID 来自服务端环境变量；用户角色只由权限中心查询，客户端不能自行声明角色。

该示例固定发送 `authorization_type: "api"`。`attributes` 是可选字段：只有配置的 claim 在已验证 JWT 中存在时才发送；没有可用属性时不会发送空对象。若接入对象级数据授权，应由服务端为对应路由固定发送 `data`，并从可信数据源补充 `resource.attributes`，不能让浏览器选择鉴权类型或伪造业务对象属性。
