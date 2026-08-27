# Permission Center

一个使用 Go、PostgreSQL 和 React 实现的 RBAC 权限中心。服务接入 NexusAuth OAuth 2.0 / OpenID Connect 统一登录，采用后端管理的加密 HttpOnly Cookie 会话，浏览器不保存 access token。

## 模型

- `application`：不是表，也不是可管理的实体。它只是 `roles` 和 `resources` 表上的字符串标识字段，例如 `admin-console`，用于隔离数据。
- `role`：角色 ID 使用 `VARCHAR(80)` 字符串，角色编码在同一 `application` 内唯一。
- `resource`：同一 `application` 内的权限资源树。`type=menu` 用于导航节点；`type=button` 用于页面操作，必须挂在菜单下，并提供受保护的 `api_path`（可选 `http_method`）。资源编码在同一应用内唯一。
- `role_resource`：角色和资源的多对多授权关系。一次 `PUT` 会在单一事务中全量替换该角色的授权集。
- `user_role`：NexusAuth 用户 `sub` 与角色的多对多关系。`subject` 和 `role_id` 均为 `VARCHAR(80)`；不在本服务复制用户表。分配角色时按应用全量替换，其他应用的角色保持不变。

四张 RBAC 表统一使用公共审计字段：`created_by_id`、`created_by_name`、`created_at`、`updated_by_id`、`updated_by_name`、`updated_at`、`is_deleted`。创建人与修改人由后端从 NexusAuth 会话读取，接口不接受客户端传入审计身份；OIDC 关闭时记录为 `development / 开发模式`。删除及关联关系替换均使用 `is_deleted` 软删除，查询只返回未删除数据。

应用隔离是最重要的约束：角色不能关联其他应用的资源，资源父节点也不能跨应用。资源和授权在查询时按 `sort, created_at` 保持稳定顺序。

## API

| Method | Path | Purpose |
| --- | --- | --- |
| POST/GET | `/v1/roles` | 创建角色、按 `application` 查询角色 |
| POST | `/v1/resources` | 创建菜单或按钮资源 |
| GET | `/v1/resources/tree` | 按 `application` 获取资源树 |
| PUT | `/v1/roles/{roleID}/resources` | 全量替换角色资源授权 |
| GET | `/v1/roles/{roleID}/resources` | 获取角色拥有的资源 ID |
| PUT | `/v1/users/{userID}/roles?application={application}` | 全量替换该 NexusAuth 用户在指定应用的角色 |
| GET | `/v1/users/{userID}/roles?application={application}` | 获取该用户在指定应用的角色 ID |

启用 OIDC 后，上述 `/v1/*` 接口全部要求登录。`/healthz` 与以下认证接口公开：

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/auth/config` | 获取不含密钥的 OIDC 公共配置 |
| GET | `/auth/login` | 创建 state、nonce、PKCE 并返回 NexusAuth 授权地址 |
| GET | `/auth/callback` | 交换 code、验证 ID Token 并建立会话 |
| GET | `/auth/me` | 获取当前登录用户 |
| POST | `/auth/logout` | 清除会话并返回 NexusAuth 登出地址 |

## NexusAuth 接入

在 NexusAuth 中自行创建客户端和服务资源并完成绑定。开发环境建议登记：

- Client ID：`permission-center-web`，也可以使用你的实际标识。
- 客户端认证方式：`client_secret_basic`。
- Redirect URI：`http://localhost:5273/api/auth/callback`。
- Post logout redirect URI：`http://localhost:5273/`。
- 服务资源 Scope：示例为 `permission-center-api`。
- 允许的 Scope：`openid profile email offline_access permission-center-api`。

修改 `configs/app.yaml` 中的 `oidc` 配置，生成独立的随机 `session_secret`，最后设置 `enabled: true`。对应环境变量均以 `PERMISSION_CENTER_OIDC_` 开头，例如 `CLIENT_ID`、`CLIENT_SECRET`、`SCOPES` 和 `SESSION_SECRET`。

`oidc.enabled=false` 仅用于尚未注册 NexusAuth 客户端时的本地开发，此时 API 会绕过登录保护。

请求示例：

```sh
curl -X POST http://localhost:8080/v1/roles \
  -H 'Content-Type: application/json' \
  -d '{"application":"admin-console","code":"operator","name":"运营人员"}'

curl -X POST http://localhost:8080/v1/resources \
  -H 'Content-Type: application/json' \
  -d '{"application":"admin-console","code":"system:user","name":"用户管理","type":"menu","path":"/system/users","sort":10}'

curl -X POST http://localhost:8080/v1/resources \
  -H 'Content-Type: application/json' \
  -d '{"application":"admin-console","parent_id":"'$MENU_ID'","code":"system:user:create","name":"新增用户","type":"button","api_path":"/api/users","http_method":"POST"}'
```

## Run

```sh
docker compose up -d postgres
./scripts/migrate.sh
go run ./cmd/api-server

cd dashboard
yarn install
yarn dev
```

访问 `http://localhost:5273/`。可用 `PERMISSION_CENTER_DATABASE_URL` 和 `PERMISSION_CENTER_HTTP_ADDR` 覆盖数据库及监听配置。本项目不使用消息队列，授权变更直接通过事务写入 PostgreSQL。
