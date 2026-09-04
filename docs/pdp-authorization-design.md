# PDP 业务授权设计

## 1. 目标与边界

权限中心当前已经具备应用隔离的 RBAC：`user_roles` 将 NexusAuth 的 `subject` 关联到角色，`role_menus` 将角色关联到菜单与按钮。它适合回答“登录后能看到哪些页面、能使用哪些操作”。

本设计新增通用 PDP（Policy Decision Point），用于回答业务请求是否允许执行：

- 接口：是否允许调用一个 HTTP API。
- 业务资源：是否允许读取、修改、删除某一条业务数据，例如仅修改自己发布的帖子。
- 数据范围与字段：列表可见哪些行，以及字段是隐藏、只读、可写还是脱敏。
- 菜单与按钮：根据现有角色授权返回可见的导航与操作节点。
- 限流：允许请求时同时返回必须执行的限流义务（obligation）。

NexusAuth 只提供认证身份（OIDC `sub`、必要的 claims）；它不参与任何授权计算。业务服务内的 PEP（Policy Enforcement Point）负责收集上下文并向本服务的 PDP 请求决策，然后执行结果。

> 实施状态（Phase A，已完成）：资源、动作、API 端点、策略、角色/subject
> 绑定、策略模拟和 `POST /v1/pdp/decisions` 已实现。首期资源类型仅支持
> `api` 与 `entity`，选择器支持精确编码与 `*`，决策采用 deny-overrides 和
> default deny。CEL、ReBAC、字段义务、数据范围和限流义务仍属于后续阶段。

本方案不把业务数据同步进权限中心，不引入 Kafka 或消息队列，也不复用 `menus` 作为业务资源表。

## 2. 总体架构

```text
Browser / Service Client
          |
          v
Business API / Gateway (PEP) ---- read business object / claims ---- PIP
          |                                                         |
          | POST /v1/pdp/decisions                                 |
          v                                                         v
Permission Center PDP ---- PostgreSQL: roles, resources, policies, bindings
          |
          +---- decision: allow / deny + matched policies + obligations
```

- **PEP**：在网关或业务 API 中。认证完成后取得可信 `subject`，解析路由、HTTP 方法和业务对象属性；PDP 返回结果后实际放行、拒绝或执行限流。
- **PDP**：权限中心中的纯决策服务。它不直接调用业务数据库，不信任浏览器提交的用户 ID、角色或资源 owner。
- **PIP**：属性来源。第一阶段由 PEP 直接提供最小业务属性；后续可接入只读属性适配器（例如组织、租户、数据 owner），但不跨服务写数据。
- **PAP**：权限中心管理端，用于维护资源、动作、策略和策略绑定。

## 3. 与现有 RBAC 的关系

| 能力 | 现有数据 | PDP 扩展后的职责 |
| --- | --- | --- |
| 菜单、按钮可见性 | `roles`、`menus`、`role_menus`、`user_roles` | 保持现有模型，新增“按 subject 返回已授权树”的查询接口 |
| 粗粒度 API 权限 | 菜单按钮的 `api_path`、`http_method` | 可迁移为 `api` 类型资源和动作；菜单按钮仍仅是 UI 节点 |
| 数据归属/租户/状态 | 无 | 通过 ABAC 条件引用 `subject`、`resource` 与 `environment` 属性 |
| 分享、协作、owner 等关系 | 无 | ReBAC 关系元组描述实例与用户/组之间的关系 |
| 列表、字段展示 | 无 | 返回声明式行级过滤和字段义务，由业务查询层执行 |
| 限流 | 无 | 策略返回限流 obligation，PEP 使用本地或共享限流器执行 |

角色仍是最常见的主体集合，策略绑定可以直接绑定角色。需要对单个用户临时授权时再绑定 `subject`，不要通过创建一次性角色解决。

## 4. 核心领域模型

所有可变配置表统一使用当前项目的基础审计字段：`created_by_id`、`created_by_name`、`created_at`、`updated_by_id`、`updated_by_name`、`updated_at`、`is_deleted`。业务主键不使用 UUID 的场景继续遵循 `VARCHAR(80)` 约束。

### 4.1 `authorization_resources`

资源注册表，描述“要保护的资源类别或路由模板”，不保存帖子、订单等实例数据。资源命名空间使用 `application.code`，例如 `forum.post`、`crm.order`，防止跨应用重名。

| 字段 | 说明 |
| --- | --- |
| `id` | UUID 主键 |
| `application` | 应用标识 |
| `code` | 稳定资源编码，例如 `post`、`post-api` |
| `type` | Phase A 为 `api`、`entity`；`menu`、`rate_limit` 预留给后续阶段 |
| `name` / `description` | 管理展示信息 |
| `matcher` | API 资源可使用受限路径模板，如 `/v1/posts/{postId}`；实体资源为空 |
| `attribute_schema` | JSON Schema 子集，声明 PDP 可引用的实例属性，例如 `ownerId`、`tenantId`、`status` |
| `field_schema` | 字段清单与敏感级别，供策略创建时校验字段名；不保存字段值 |
| `enabled` | 是否参与决策 |

同一 `application + code` 的有效资源唯一。资源粒度为“租户 -> 应用 -> 模块 -> 资源类型 -> 实例（行）-> 字段（列）”。`menus` 不迁入该表；如需给菜单可见性统一建模，仅建立对原 `menu.code` 的引用，不复制树。

### 4.2 `authorization_actions`

动作注册表，避免把 HTTP 方法、业务动作和按钮编码混为一个字段。

| 字段 | 说明 |
| --- | --- |
| `id` | UUID 主键 |
| `application` | 应用标识 |
| `code` | 如 `read`、`create`、`update`、`delete`、`publish` |
| `name` / `description` | 展示信息 |
| `enabled` | 是否参与决策 |

同一应用内动作编码唯一。API 类型资源的 PEP 可将 `GET` 映射到 `read`、`POST` 映射到 `create`，映射显式配置在 PEP 或 API 资源元数据中，不隐式猜测。

### 4.3 `authorization_api_endpoints`

业务 API 的受保护端点注册表。它与菜单按钮完全独立：一个 API 可以没有对应按钮，一个按钮也不能因配置了 `api_path` 就自动获得后端调用权限。

| 字段 | 说明 |
| --- | --- |
| `id` | UUID 主键 |
| `application` | 应用标识 |
| `service_code` | 业务服务标识，例如 `forum-api` |
| `method` | HTTP 方法，`GET`、`POST`、`PUT`、`PATCH`、`DELETE` 等 |
| `path_template` | 经路由框架标准化后的模板，例如 `/v1/posts/{postId}`，不用原始 URL |
| `resource_id` | 对应 `authorization_resources` 中 `type=api` 或实体资源的 ID |
| `action_id` | 对应动作，例如 `post.update` 或 `update` |
| `enforcement_mode` | `enforce`、`audit`、`disabled`；先以 audit 灰度接入 |
| `enabled` | 是否生效 |

有效端点在 `application + service_code + method + path_template` 内唯一。API 资源建议分为两层：端点资源用于粗粒度接口调用权，实体资源用于单条数据 owner、tenant、状态和字段权限。一次 API 调用可连续做两次决策，例如先检查能否调用 `PUT /v1/posts/{postId}`，再检查能否更新 `post:post-42`。

### 4.4 `authorization_policies`

策略是可版本化、可启停的决策规则。

| 字段 | 说明 |
| --- | --- |
| `id` | UUID 主键 |
| `application` | 应用标识 |
| `code` / `name` | 稳定编码与名称 |
| `effect` | `allow` 或 `deny` |
| `priority` | 整数，越大优先级越高 |
| `resource_selector` | 资源编码集合或通配选择器，第一期仅支持精确编码和 `*` |
| `action_selector` | 动作编码集合或 `*` |
| `condition_expression` | CEL 表达式；空表达式表示主体绑定即可命中 |
| `obligations` | JSON 数组，例如限流义务、字段脱敏义务 |
| `version` | 递增版本，用于缓存失效和审计 |
| `enabled` | 是否生效 |

表达式可读取以下根对象，禁止任意函数、网络、文件和数据库访问：

```text
subject.id, subject.roles, subject.claims, subject.attributes
resource.type, resource.id, resource.attributes
environment.tenantId, environment.ip, environment.now
request.method, request.path
```

帖子“只能修改自己发布的内容”的示例：

```text
resource.code = post
action = update
effect = allow
condition_expression = resource.attributes.ownerId == subject.id
```

管理员的全量编辑策略可绑定 `role:post-admin`，无条件 `allow`。禁止访问的策略使用 `deny`，例如内容处于归档状态时拒绝更新。

### 4.5 `authorization_policy_bindings`

策略绑定定义该策略适用于谁。

| 字段 | 说明 |
| --- | --- |
| `policy_id` | 策略 ID |
| `subject_type` | `role`、`subject`；预留 `group`、`service` |
| `subject_value` | 角色 ID 或 NexusAuth subject，最大 80 字符 |
| `enabled` | 可暂停一条绑定 |

角色绑定必须验证角色与策略应用一致；`subject` 绑定是精确授权，不创建本地用户表。

### 4.6 `authorization_policy_versions` 与 `authorization_decision_logs`

- `authorization_policy_versions` 保存策略更新前后的不可变快照、版本、变更原因，支持回滚和审计。
- `authorization_decision_logs` 是可配置采样的决策审计：`request_id`、`application`、`subject_id`、资源/动作、结果、命中策略版本、拒绝原因、耗时、创建时间。默认不保存完整 claims、请求体或业务属性，防止敏感信息泄露。

这两个表是第一期建议实现的审计基础；日志按日期分区和保留期清理，避免无限增长。

### 4.7 `authorization_relation_tuples`

ReBAC 关系元组补足 owner、分享、协作等不能只靠角色或属性表达的授权关系。

| 字段 | 说明 |
| --- | --- |
| `application` | 应用标识 |
| `resource_code` / `resource_id` | 资源类型与实例，例如 `post` / `post-42` |
| `relation` | `owner`、`editor`、`viewer`、`shared_with` 等受资源 schema 约束的关系名 |
| `subject_type` / `subject_value` | `subject`、`role`、`group`；值为 subject/角色/组标识 |
| `expires_at` | 可选有效期，支持临时共享或临时授权 |

示例：`post:post-42#owner@subject:nexus-subject`。PDP 可在 CEL 中通过受限函数 `relation.has("owner")` 判断当前 subject 是否拥有关系。第一期只支持精确实例关系与直接 subject，组关系需要先定义可信 PIP 后再开放。

### 4.8 数据范围与字段义务

策略不返回任意 SQL 字符串，也不让 PDP 直接访问业务数据库。它返回由 PEP/业务数据层翻译的声明式义务：

```json
{
  "type": "data_scope",
  "mode": "owner",
  "ownerField": "ownerId"
}
```

```json
{
  "type": "field_access",
  "fields": {
    "title": "write",
    "authorId": "read",
    "phone": "mask",
    "internalNote": "hidden"
  }
}
```

字段状态限定为 `hidden`、`read`、`write`、`mask`、`conditional_write`。相同字段规则通过字段模板复用，避免在每一条策略中逐字段复制。

## 5. 决策语义

### 5.1 输入

PEP 调用内部接口时发送：

```json
{
  "application": "content-platform",
  "subject": { "id": "nexus-subject", "claims": { "tenantId": "t-1" } },
  "resource": {
    "code": "post",
    "type": "entity",
    "id": "post-42",
    "attributes": { "ownerId": "nexus-subject", "tenantId": "t-1", "status": "draft" }
  },
  "action": "update",
  "environment": { "tenantId": "t-1", "ip": "203.0.113.1" },
  "request": { "method": "PUT", "path": "/v1/posts/post-42", "requestId": "..." }
}
```

`subject.id` 必须由认证后的 PEP 注入。外部浏览器不得直接调用决策接口，也不得自行提交角色集合；PDP 从本地 `user_roles` 查询角色。

### 5.2 业务 API 的 PEP 执行链路

每个业务服务在认证中间件之后、业务 handler 之前挂载 PDP PEP。对一个修改帖子接口的完整执行顺序如下：

```text
1. OIDC / 网关认证中间件验证 Token 或 BFF 会话，得到可信 subject。
2. 路由框架提供标准路由模板和 path 参数：PUT /v1/posts/{postId}。
3. PEP 以 application + service_code + method + path_template 查询端点绑定。
4. PEP 调用 PDP：检查该 subject 是否可调用 post-api.update。
5. 若端点允许，业务服务根据 postId 从自己的数据库读取帖子 owner、tenant、status。
6. PEP/业务服务再次调用 PDP：检查该 subject 是否可 update 具体 post 实例。
7. 允许后执行业务写入；拒绝返回统一 403，限流 obligation 则先执行限流器。
```

端点决策请求示例：

```json
{
  "application": "forum",
  "subject": { "id": "nexus-subject" },
  "resource": {
    "code": "post-api",
    "type": "api",
    "attributes": { "serviceCode": "forum-api" }
  },
  "action": "update",
  "request": {
    "method": "PUT",
    "pathTemplate": "/v1/posts/{postId}",
    "requestId": "req-123"
  }
}
```

端点授权不等于数据授权：接口策略只回答“可不可以进入该操作”；实例策略再回答“可不可以改这条帖子”。创建操作没有已有实例，可由 PDP 根据请求中经 schema 白名单筛选后的 tenant、父资源或业务状态进行判断。不得将未经校验的请求体整体送给 PDP。

PEP 的拒绝响应统一为 HTTP `403`，业务错误码为 `AUTHORIZATION_DENIED`；未注册端点、端点 disabled、PDP 超时或策略解析失败默认拒绝。`audit` 模式记录本应拒绝的决策日志但不阻断，仅用于灰度期，必须有明确到期时间。

### 5.3 算法

1. 校验应用、资源、动作存在且启用；未注册资源或动作默认拒绝。
2. 查询该 `subject` 在该应用的有效角色；组合主体集合 `subject` 与 `role`。
3. 取有效主体绑定关联的策略，先按 `priority DESC`，再按稳定 ID 排序。
4. 解析当前资源实例的有效 ReBAC 关系；只保留资源、动作选择器匹配且 CEL 条件为 `true` 的策略。
5. **拒绝优先**：任一命中的 `deny` 策略优先于任何 `allow`。多个同 effect 的策略按优先级记录为命中原因。
6. 没有命中 `allow` 时拒绝（default deny）。
7. 合并命中 allow 策略的 obligations；同一限流 key 采用最严格规则，字段权限按“最小可见/最小可写”合并。

返回结果固定为：`allow`、`reasonCode`、`matchedPolicyIds`、`policyVersions`、`obligations` 与 `decisionId`。生产环境对调用方可隐藏具体策略名称，对审计和管理端保留完整信息。

### 5.4 限流 obligation

示例：

```json
{
  "type": "rate_limit",
  "keyTemplate": "{application}:{subject.id}:{resource.code}:{action}",
  "algorithm": "token_bucket",
  "limit": 20,
  "windowSeconds": 60,
  "onExceeded": "deny"
}
```

PDP 只决定是否附带该义务；真正计数在 PEP 执行。单实例可使用内存令牌桶，分布式服务必须使用 Redis 或网关原生限流能力。PostgreSQL 负责配置与审计，不应成为高频限流计数器；这与“不使用消息队列”没有冲突。

### 5.5 列表与字段执行

列表页不得把整批数据查出后逐行请求 PDP，也不能在应用内存中过滤。业务服务应将 `data_scope` 翻译为参数化查询条件，例如 owner 范围下推为 `WHERE owner_id = :subject_id`，再在序列化响应前执行字段隐藏/脱敏。这样分页总数、排序和导出结果都在同一数据范围中，避免越权泄露。

单个资源详情、更新和删除仍必须先请求 PDP；查询过滤只是列表读路径的补充，不能替代服务端 PEP。

## 6. 对外接口草案

接口继续返回当前项目统一响应信封。

| Method | Path | 用途 |
| --- | --- | --- |
| `POST` | `/v1/pdp/decisions` | 单次授权决策，供受信任 PEP 调用 |
| `POST` | `/v1/pdp/decisions:batch` | 同一 subject 的批量菜单/按钮或列表决策 |
| `POST` | `/v1/pdp/data-scopes` | 返回可翻译的行级过滤与字段义务，不返回 SQL |
| `POST` | `/v1/authorization/resources` | 创建资源定义 |
| `GET/PUT/DELETE` | `/v1/authorization/resources/{id}` | 资源定义管理 |
| `POST` | `/v1/authorization/actions` | 创建动作 |
| `GET/PUT/DELETE` | `/v1/authorization/actions/{id}` | 动作管理 |
| `POST` | `/v1/authorization/api-endpoints` | 注册业务 API 与资源/动作绑定 |
| `GET/PUT/DELETE` | `/v1/authorization/api-endpoints/{id}` | 端点绑定管理、灰度模式切换 |
| `POST` | `/v1/authorization/policies` | 创建草稿策略 |
| `GET/PUT/DELETE` | `/v1/authorization/policies/{id}` | 策略管理、启停、软删除 |
| `PUT` | `/v1/authorization/policies/{id}/bindings` | 原子全量替换主体绑定 |
| `POST` | `/v1/authorization/policies/{id}:simulate` | 使用指定上下文模拟，不写决策日志 |
| `GET` | `/v1/navigation?application=...` | 根据当前 subject 返回过滤后的既有菜单/按钮树 |

决策接口要使用服务间认证（mTLS、网关签名 JWT 或专用 client credential），不能仅依赖浏览器 Cookie。管理接口沿用当前 NexusAuth BFF 会话，但还需要一个“权限中心管理员”初始化策略，避免任意已登录用户修改策略。

## 7. 缓存、一致性与安全

- 缓存的键必须包含 `application + subject + resource.code + action + policyVersion + 属性摘要`。有 `resource.id` 或 owner 条件时不能只缓存“用户有 update 权限”。
- 策略、绑定、用户角色变化时，事务提交后提升应用级 `authorization_revision`。PDP 节点按 revision 失效本地缓存；第一期单进程可直接清空缓存。
- 决策服务无状态、可水平扩展；目标为单资源决策 P99 小于 10ms。列表路径使用数据范围决策，不能对每一行逐条决策。
- 未注册资源、解析失败、CEL 类型错误、PIP 缺少必要属性、限流器不可用时默认拒绝（fail closed），健康检查与明确白名单除外。
- 条件表达式创建/更新时编译并校验，决策路径只执行已验证表达式；限制表达式大小、嵌套、执行时间和允许函数。
- 审计日志记录 request ID 与策略版本，不记录 access token、Cookie、完整请求体及敏感 claims。

## 8. 分阶段实施

### 阶段 A：PDP 最小闭环

1. 已新增资源、动作、API 端点、策略和绑定表及审计字段；策略版本表保留到运行治理阶段。
2. 新增 API 端点注册表；实现 RBAC 主体绑定、精确资源/动作选择、deny-overrides、default deny。
3. 已实现 `/v1/pdp/decisions`、策略模拟、单元测试、SDK 测试和管理端端到端测试。
4. 已提供 Go HTTP SDK 的 `Decide`、`Check` 与 `AuthorizeHTTP`；第一个真实业务 API 的 PEP 接入仍由业务服务完成，不改动现有菜单模型。

### 阶段 B：ABAC 与业务接入

1. 引入 CEL 条件、资源属性 schema、ReBAC 关系元组和 owner/tenant 示例。
2. 为一个真实业务服务接入 PEP，业务服务从自身数据库读取 owner 等属性，并将列表数据范围下推到 SQL。
3. 实现字段模板、字段义务与敏感字段脱敏；增加按 subject 的菜单/按钮过滤 API，前端只渲染该接口返回的树。

### 阶段 C：限流与运行治理

1. 支持 rate-limit obligation 与 PEP 限流适配器。
2. 加入决策采样、策略版本回滚、缓存 revision、监控指标。
3. 再评估 group、组织层级、字段脱敏等复杂 obligation，避免过早扩张 DSL。

## 9. 本次评审需要确认的决策

1. 条件语言采用 CEL，还是团队已有的表达式标准；建议 CEL，因为 Go 生态成熟且可静态校验。
2. 决策接口的服务间认证方式：mTLS、网关签名 JWT，或 NexusAuth client credentials。建议网关签名 JWT 或 mTLS，避免把浏览器身份透传为 PDP 调用凭据。
3. 限流的运行组件：单实例内存、Redis，还是已有 API Gateway。策略配置在本服务不受此选择影响。
4. 第一个接入 PDP 的业务资源：建议选择“帖子 owner 可修改”或一个真实 API，作为 ABAC 验收样例。

## 10. 参考方案对比

参考讨论提出的“RBAC + ABAC + ReBAC”和“接口门 + 数据门”分层与本方案一致，已采纳 ReBAC 关系元组、字段义务、数据范围下推和性能预算。

Casbin 可以作为业务服务内接口级 PEP 的轻量适配器，但不作为权限中心 PDP 的核心：它无法原生解决关系元组、字段义务、策略版本仿真和查询过滤。若后续接入方已使用 Casbin，可由 SDK 将 PDP 的 allow/deny 决策映射到其第一道接口门，数据范围和字段义务仍由业务数据层执行。

确认以上四点后，实施将从阶段 A 开始，并补充数据库迁移、管理端页面、PDP API、PEP 示例及端到端测试。
