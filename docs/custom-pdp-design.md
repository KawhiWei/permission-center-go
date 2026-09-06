# 自研 PDP 授权引擎设计

## 1. 目标与边界

本项目实现自有的策略决策点（PDP），不集成 OPA，不使用 Rego，不部署 OPA 服务。参考 OPA 的是架构原则：策略与业务代码解耦、PEP 和 PDP 职责分离、结构化决策输入输出、默认拒绝和可审计。

当前范围包含 API 接口级授权和单个业务对象的数据级 ABAC 授权。`tenant_id` 是决策输入的必填安全边界，不得由请求参数直接信任。

非目标：

- 菜单和按钮不参与 API 授权决策。
- PDP 不查询业务数据库，不拥有业务行数据。
- 不生成批量查询的行过滤 SQL，也不返回字段脱敏规则。
- 不允许策略中存储 Go、JavaScript、SQL 或其他可执行脚本。

## 2. 责任分层

```text
管理员 -> 权限中心策略控制面 -> 策略版本/发布/审计
                                      |
                                      v
                              不可变决策快照
                                      |
业务请求 -> PEP 中间件 -> 自研 PDP Go 评估器 -> Decision
                 |                                      |
                 +-- 身份/租户/资源属性 --------+
```

### 控制面

权限中心负责策略 CRUD、结构校验、模拟测试、版本发布、回滚、快照生成和审计。草稿不得被运行时评估。

### PEP

PEP 是业务 API 的必经入口，可直接通过 HTTP 调用 PDP。PEP 负责：

1. 从已验证的会话或 token 取得 subject 和 tenant，不信任客户端自报值。
2. 使用路由模板而不是原始 URL 定位已登记 API 端点。
3. 收集决策所需的最小上下文并请求 PDP。
4. `deny` 返回 HTTP 403；PDP 不可用时默认失败关闭。
5. 根据路由场景固定选择 `api` 或 `data`，不能让浏览器自由指定鉴权类型。

### PDP

PDP 是纯决策运行时：加载已发布快照，根据输入匹配策略，使用确定性组合算法返回结果。PDP 不修改策略，不查业务数据，不根据菜单授权。

## 3. 菜单权限与 API 权限

```text
role_menus                     policy_role_bindings
    |                                  |
    v                                  v
菜单/按钮可见性                    authorization_policies
                                       |
                                       v
                              policy_api_targets -> API endpoint
```

- `role_menus` 只用于导航和操作按钮可见性，它不是安全边界。
- 按钮保留稳定 `code`，并可配置一组 `api_path` 和 `http_method`，用于描述该 UI 操作通常触发的单个 API。它们是界面元数据和管理辅助引用，不是 PDP 的授权事实来源，也不建立菜单与授权策略的关联。
- API 端点是 PDP 的资源对象。策略分别关联角色和 API 端点。
- API 与角色不建立直接绑定，PDP 只读取已发布策略中的角色和端点目标。

## 4. 枚举与类型

Go 中全部使用带类型的 string enum 并实现 `Validate()`。PostgreSQL 使用同值 `CHECK` 约束，避免原生 ENUM 删值和回滚困难，但禁止无约束字符串。

| 类型 | 初始枚举 |
| --- | --- |
| `PolicyEffect` | `allow`, `deny` |
| `PolicyStatus` | `draft`, `published`, `disabled`, `archived` |
| `AuthorizationType` | `api`, `data` |
| `TargetKind` | `api_endpoint` |
| `CombiningAlgorithm` | `deny_overrides` |
| `Decision` | `allow`, `deny` |
| `ValueSource` | `subject`, `resource`, `request`, `context`, `literal` |
| `ValueType` | `string`, `number`, `boolean`, `timestamp`, `string_list`, `number_list` |
| `LogicalOperator` | `all`, `any`, `not` |
| `ComparisonOperator` | `eq`, `neq`, `in`, `not_in`, `contains`, `exists`, `lt`, `lte`, `gt`, `gte`, `starts_with`, `ends_with` |
| `ReasonCode` | `allowed_by_policy`, `explicit_deny`, `default_deny`, `scope_mismatch`, `endpoint_disabled`, `invalid_input`, `engine_unavailable` |

只开放实际需要的操作符。新增操作符必须同时提供类型检查、评估器测试和负向用例，不支持任意函数调用。

## 5. 数据模型

### 必需表

`authorization_policies`

- `id uuid`
- `service_resource varchar(128)`
- `code varchar(160)`，服务资源内唯一
- `name`, `description`
- `effect PolicyEffect`
- `status PolicyStatus`
- `authorization_type AuthorizationType`，`api` 表示接口级授权，`data` 表示针对具体业务数据属性的对象级授权
- `priority integer`
- `condition jsonb`，结构化 DSL AST
- `current_version integer`
- 公共审计与软删除字段

`authorization_policy_role_bindings`

- `policy_id uuid`
- `role_id varchar(80)`
- 联合主键 `(policy_id, role_id)`
- 数据库触发器校验策略和角色属于同一 `service_resource`

`authorization_policy_api_targets`

- `policy_id uuid`
- `endpoint_id uuid`
- 联合主键 `(policy_id, endpoint_id)`
- 数据库触发器校验策略和端点属于同一 `service_resource`

`authorization_policy_versions`

- `policy_id uuid`, `version integer`
- `snapshot jsonb`，完整不可变策略快照
- `checksum varchar(64)`
- `published_by`, `published_at`
- 联合主键 `(policy_id, version)`

`authorization_decision_logs`

- `decision_id uuid`, `request_id varchar(80)`
- `service_resource`, `tenant_id`, `subject_id`
- `endpoint_id`, `decision`, `reason_code`
- `matched_policy_ids uuid[]`, `policy_snapshot_version`
- `latency_ms`, `occurred_at`
- 不记录 token、密钥和完整敏感业务属性

## 6. 结构化策略 DSL

策略条件是受 schema 约束的 AST，不是文本脚本。示例：

```json
{
  "all": [
    {
      "left": { "source": "subject", "path": "tenant_id", "type": "string" },
      "op": "eq",
      "right": { "source": "resource", "path": "tenant_id", "type": "string" }
    },
    {
      "left": { "source": "subject", "path": "account_locked", "type": "boolean" },
      "op": "eq",
      "right": { "source": "literal", "value": false, "type": "boolean" }
    }
  ]
}
```

校验器必须限制：最大嵌套深度、节点数、列表长度、属性路径白名单、操作符与值类型组合。数值比较不得隐式转换字符串。

## 7. 决策接口

`POST /v1/pdp/decisions`

决策请求使用必填的 `authorization_type` 选择运行策略：`api` 判断是否允许访问 API 端点，`data` 结合当前业务对象的属性进行 allow/deny 判断。字段缺失或值未知时直接拒绝。`attributes` 保持可选，只有被策略条件引用时才需要传入，缺失属性不会匹配比较条件。

```json
{
  "request_id": "req-01",
  "authorization_type": "data",
  "service_resource": "order-service",
  "tenant_id": "tenant-a",
  "subject": {
    "id": "user-1",
    "attributes": { "account_locked": false, "department": "sales" }
  },
  "action": { "kind": "http", "method": "GET" },
  "resource": {
    "kind": "api_endpoint",
    "endpoint_id": "00000000-0000-0000-0000-000000000001",
    "attributes": { "tenant_id": "tenant-a" }
  },
  "context": { "time": "2026-09-05T10:00:00Z", "ip": "192.0.2.1" }
}
```

```json
{
  "decision_id": "00000000-0000-0000-0000-000000000002",
  "decision": "allow",
  "reason_code": "allowed_by_policy",
  "matched_policy_ids": ["00000000-0000-0000-0000-000000000003"],
  "snapshot_version": 12
}
```

决策请求不接受 `role_ids`。权限中心根据已验证 subject 和 service resource 加载有效角色，再生成只供评估器使用的内部输入。

## 8. 评估算法

1. 校验输入和服务资源边界，失败返回 `deny/invalid_input`。
2. 校验 API 端点存在、启用且属于当前服务资源。
3. 以 subject 的有效角色和 endpoint 筛选已发布候选策略。
4. 按快照版本执行类型安全条件评估。
5. 任一匹配 `deny` 策略则拒绝。
6. 无 deny 且至少一条 `allow` 匹配则允许。
7. 其余情况统一 `deny/default_deny`。

`priority` 用于稳定评估顺序和解释，不得让高优先级 allow 覆盖 deny。当前组合算法只开放 `deny_overrides`。

## 9. 运行时与安全

- PDP 由权限中心作为集中式 HTTP 服务提供，评估核心保持无 I/O 的 Go 包。
- PDP 决策 API 只接受携带服务凭据的 PEP 调用，不向浏览器公开。
- 决策请求只读取当前发布版本对应的不可变快照，不读取草稿内容。
- 决策超时、快照缺失、属性类型错误和未知操作符均失败关闭。
- 决策日志与业务审计日志分表，设置保留周期和敏感字段脱敏。
- 对发布、回滚、模拟决策和实时决策分别授权，禁止用策略管理权限隐式获得业务 API 权限。

## 10. Go 模块边界

```text
internal/policy/model       枚举、DSL AST、Decision 契约
internal/policy/validate    策略和输入类型校验
internal/policy/compile     草稿 -> 不可变快照
internal/policy/eval        纯函数评估器
internal/biz/policy         策略生命周期与发布
internal/data/db/repo       策略、绑定、版本、日志仓储
internal/server/http        管理 API 与 PDP 决策 API
sdk/pdp                     PEP client、middleware、超时与 fail-closed
```

`internal/policy/eval` 不得依赖 HTTP、PostgreSQL 或具体业务模型，保证可以用表驱动用例、fuzz 和 benchmark 独立验证。

## 11. 验收底线

- 没有匹配 allow 时必须 deny。
- 任一匹配 deny 不得被 allow 覆盖。
- 菜单或按钮授权不得改变 API 决策。
- 不同 `service_resource` 和 `tenant_id` 之间无法关联策略、角色和 target。
- 相同快照和输入始终产生相同决策。
- 发布失败不得污染当前运行快照。

## 12. 参考原则

- [AWS: 使用 OPA 实现 PDP](https://docs.aws.amazon.com/zh_cn/prescriptive-guidance/latest/saas-multitenant-api-access-authorization/opa.html)
- [AWS: 实施 PEP](https://docs.aws.amazon.com/zh_cn/prescriptive-guidance/latest/saas-multitenant-api-access-authorization/pep.html)
- [AWS: OPA 多租户设计注意事项](https://docs.aws.amazon.com/zh_cn/prescriptive-guidance/latest/saas-multitenant-api-access-authorization/opa-design-considerations.html)

这些资料只用于借鉴 PDP/PEP 责任分离和多租户隔离原则，本方案的策略模型、DSL、编译器和评估器均由本项目自行实现。
