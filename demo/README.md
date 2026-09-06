# 权限中心接入 Demo

这里提供两种不依赖权限中心 SDK、直接通过 HTTP 调用 PDP 的业务服务示例：

- [`go`](./go/README.md)：Go `net/http` + `coreos/go-oidc`。
- [`dotnet`](./dotnet/README.md)：ASP.NET Core 8 + JWT Bearer + `HttpClient`。

两个项目都演示同一个受保护接口 `GET /api/orders/{id}`，并遵循相同的安全边界：

1. 业务服务验证客户端的 OIDC Bearer JWT，从已验证 claims 提取 `sub` 和 `tenant_id`。
2. endpoint UUID 由业务服务端配置，不接受客户端指定。
3. 业务服务使用独立服务凭据请求权限中心 `POST /v1/pdp/decisions`。
4. 只有 `allow` 才执行业务逻辑；`deny` 或任何异常都不会放行。

具体环境变量和联调步骤见各项目 README。
