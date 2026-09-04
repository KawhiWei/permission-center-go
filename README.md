# Permission Center

一个使用 Go、PostgreSQL 和 React 实现的 RBAC 权限中心。服务接入 NexusAuth OAuth 2.0 / OpenID Connect 统一登录，采用后端管理的加密 HttpOnly Cookie 会话，浏览器不保存 access token。

## 模型

- `service_resource`：NexusAuth 服务资源 `name` 是权限中心的统一隔离键，例如 `content-platform`。它来自 NexusAuth 开放目录，不在本服务维护生命周期。
- `role`：角色 ID 使用 `VARCHAR(80)` 字符串，角色编码在同一服务资源内唯一。
- `menu`：同一服务资源内的菜单与按钮树。`type=menu` 用于导航节点；`type=button` 用于页面操作，必须挂在菜单下，并提供受保护的 `api_path`（可选 `http_method`）。菜单项编码在同一服务资源内唯一。
- `role_menus`：角色和菜单项的多对多授权关系。一次 `PUT` 会在单一事务中全量替换该角色的授权集。
- `user_roles`：NexusAuth 用户 `sub` 与角色的多对多关系。`subject` 和 `role_id` 均为 `VARCHAR(80)`；不在本服务复制用户表。分配角色时按服务资源全量替换，其他服务资源的角色保持不变。

四张 RBAC 表统一使用公共审计字段：`created_by_id`、`created_by_name`、`created_at`、`updated_by_id`、`updated_by_name`、`updated_at`、`is_deleted`。创建人与修改人由后端从 NexusAuth 会话读取，接口不接受客户端传入审计身份；OIDC 关闭时记录为 `development / 开发模式`。删除及关联关系替换均使用 `is_deleted` 软删除，查询只返回未删除数据。

服务资源隔离是最重要的约束：角色不能关联其他服务资源的菜单项，菜单父节点也不能跨服务资源。菜单树和授权在查询时按 `sort, created_at` 保持稳定顺序。当前 `menu` 模型只负责 RBAC 菜单与按钮管理；PDP 的资源、动作、端点和策略也按同一服务资源隔离。

## API

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/v1/service-resources` | 获取 NexusAuth 服务资源目录 |
| GET | `/v1/service-resources/{name}` | 获取单个服务资源 |
| POST/GET | `/v1/roles` | 创建角色、按 `service_resource` 查询角色 |
| POST | `/v1/menus` | 创建菜单或按钮 |
| GET | `/v1/menus/tree` | 按 `service_resource` 获取菜单树 |
| PUT | `/v1/roles/{roleID}/menus` | 全量替换角色菜单授权 |
| GET | `/v1/roles/{roleID}/menus` | 获取角色拥有的菜单 ID |
| PUT | `/v1/users/{userID}/roles?service_resource={name}` | 全量替换该 NexusAuth 用户在指定服务资源的角色 |
| GET | `/v1/users/{userID}/roles?service_resource={name}` | 获取该用户在指定服务资源的角色 ID |

启用 OIDC 后，上述 `/v1/*` 接口全部要求登录。`/healthz` 与以下认证接口公开：

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/auth/config` | 获取不含密钥的 OIDC 公共配置 |
| GET | `/auth/login` | 创建 state、nonce、PKCE 并返回 NexusAuth 授权地址 |
| GET | `/signin-oidc` | 交换 code、验证 ID Token 并建立会话 |
| GET | `/auth/me` | 获取当前登录用户 |
| POST | `/auth/logout` | 清除会话并返回 NexusAuth 登出地址 |

## NexusAuth 接入

权限中心通过 NexusAuth 的 `GET /openapi/v1/service-resources` 获取可选服务资源。先在 NexusAuth Workbench 创建 `targetType=service_resource` 的开放 API 凭据，再仅在权限中心后端配置其明文 token：

```dotenv
PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_SOURCE=nexusauth
PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_NEXUSAUTH_BASE_URL=http://host.docker.internal:5100
PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_API_KEY=<service_resource-open-api-token>
```

该 token 不是 OIDC Client Secret，不能写入前端或提交到 Git。服务资源目录只读；登录后的用户必须先选择服务资源，角色、菜单、用户角色绑定和 PDP 页面才能进入。旧 `application` 请求字段仍会在过渡期被接受，但服务端返回和新前端请求均使用 `service_resource`。

在 NexusAuth 中自行创建客户端和服务资源并完成绑定。开发环境建议登记：

- Client ID：`permission-center-web`，也可以使用你的实际标识。
- 客户端认证方式：`client_secret_basic`。
- **Redirect URI（回调地址）**：`http://localhost:8080/signin-oidc`。
- **Post logout redirect URI（登出回跳地址）**：`http://localhost:5274/`。
- 服务资源 Scope：示例为 `permission-center-api`。
- 允许的 Scope：`openid profile email offline_access permission-center-api`。

本地 NexusAuth 的 Authority 为 `http://localhost:5100`。实现与 NexusAuth Workbench 的 BFF 模式一致：NexusAuth 直连 Go 后端回调地址，后端建立加密会话后跳转到 `http://localhost:5274/auth/callback`。服务从 OIDC discovery 文档自动读取统一登出端点 `http://localhost:5100/connect/endsession`；浏览器点击“退出登录”后，后端先清除权限中心会话，再携带 `id_token_hint` 和上述登出回跳地址跳转至该端点。

修改 `configs/app.yaml` 中的 `oidc` 配置，生成独立的随机 `session_secret`，最后设置 `enabled: true`。对应环境变量均以 `PERMISSION_CENTER_OIDC_` 开头，例如 `CLIENT_ID`、`CLIENT_SECRET`、`SCOPES` 和 `SESSION_SECRET`。容器部署时，`BACKCHANNEL_AUTHORITY` 用于服务端 discovery、Token 和 JWKS 请求，浏览器跳转仍使用公开的 `AUTHORITY`。

`oidc.enabled=false` 仅用于尚未注册 NexusAuth 客户端时的本地开发，此时 API 会绕过登录保护。

请求示例：

```sh
curl -X POST http://localhost:8080/v1/roles \
  -H 'Content-Type: application/json' \
  -d '{"application":"admin-console","code":"operator","name":"运营人员"}'

curl -X POST http://localhost:8080/v1/menus \
  -H 'Content-Type: application/json' \
  -d '{"application":"admin-console","code":"system:user","name":"用户管理","type":"menu","path":"/system/users","sort":10}'

curl -X POST http://localhost:8080/v1/menus \
  -H 'Content-Type: application/json' \
  -d '{"application":"admin-console","parent_id":"'$MENU_ID'","code":"system:user:create","name":"新增用户","type":"button","api_path":"/api/users","http_method":"POST"}'
```

## Run

```sh
docker compose up -d --build
```

Compose 会依次启动 PostgreSQL、执行数据库迁移、启动 Go API 和 Nginx Dashboard。前后端均使用镜像内构建产物，不挂载宿主机源码；数据库数据保存在 Docker named volume `permission-center-data` 中。

启用统一登录时，在项目根目录创建不提交到 Git 的 `.env`：

```dotenv
PERMISSION_CENTER_OIDC_ENABLED=true
PERMISSION_CENTER_OIDC_AUTHORITY=http://localhost:5100
PERMISSION_CENTER_OIDC_BACKCHANNEL_AUTHORITY=http://host.docker.internal:5100
PERMISSION_CENTER_OIDC_CLIENT_ID=permission-center-api
PERMISSION_CENTER_OIDC_CLIENT_SECRET=<client-secret>
PERMISSION_CENTER_OIDC_SESSION_SECRET=<至少32字符的随机值>
```

访问 `http://localhost:5274/`，API 为 `http://localhost:8080`，PostgreSQL 宿主机端口为 `55433`。本项目不使用消息队列，授权变更直接通过事务写入 PostgreSQL。
