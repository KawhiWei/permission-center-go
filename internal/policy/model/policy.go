// Package model 定义与数据库实现无关、可序列化的 PDP 策略契约。
package model

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

func (v Effect) Valid() bool { return v == EffectAllow || v == EffectDeny }

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusDisabled  Status = "disabled"
	StatusArchived  Status = "archived"
)

func (v Status) Valid() bool {
	return v == StatusDraft || v == StatusPublished || v == StatusDisabled || v == StatusArchived
}

type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

type AuthorizationType string

const (
	AuthorizationTypeAPI  AuthorizationType = "api"
	AuthorizationTypeData AuthorizationType = "data"
)

func (v AuthorizationType) Valid() bool {
	return v == AuthorizationTypeAPI || v == AuthorizationTypeData
}

type ReasonCode string

const (
	ReasonAllowedByPolicy   ReasonCode = "allowed_by_policy"
	ReasonExplicitDeny      ReasonCode = "explicit_deny"
	ReasonDefaultDeny       ReasonCode = "default_deny"
	ReasonScopeMismatch     ReasonCode = "scope_mismatch"
	ReasonEndpointDisabled  ReasonCode = "endpoint_disabled"
	ReasonInvalidInput      ReasonCode = "invalid_input"
	ReasonEngineUnavailable ReasonCode = "engine_unavailable"
)

type TargetKind string

const (
	TargetAPIEndpoint TargetKind = "api_endpoint"
)

func (v TargetKind) Valid() bool { return v == TargetAPIEndpoint }

type ActionKind string

const ActionHTTP ActionKind = "http"

func (v ActionKind) Valid() bool { return v == ActionHTTP }

type ValueSource string

const (
	SourceSubject  ValueSource = "subject"
	SourceResource ValueSource = "resource"
	SourceRequest  ValueSource = "request"
	SourceContext  ValueSource = "context"
	SourceLiteral  ValueSource = "literal"
)

func (v ValueSource) Valid() bool {
	switch v {
	case SourceSubject, SourceResource, SourceRequest, SourceContext, SourceLiteral:
		return true
	}
	return false
}

type ValueType string

const (
	TypeString     ValueType = "string"
	TypeNumber     ValueType = "number"
	TypeBoolean    ValueType = "boolean"
	TypeStringList ValueType = "string_list"
	TypeNumberList ValueType = "number_list"
)

func (v ValueType) Valid() bool {
	switch v {
	case TypeString, TypeNumber, TypeBoolean, TypeStringList, TypeNumberList:
		return true
	}
	return false
}

type ComparisonOperator string

const (
	OpEq       ComparisonOperator = "eq"
	OpNeq      ComparisonOperator = "neq"
	OpIn       ComparisonOperator = "in"
	OpNotIn    ComparisonOperator = "not_in"
	OpContains ComparisonOperator = "contains"
	OpExists   ComparisonOperator = "exists"
)

func (v ComparisonOperator) Valid() bool {
	switch v {
	case OpEq, OpNeq, OpIn, OpNotIn, OpContains, OpExists:
		return true
	}
	return false
}

// ValueRef 表示条件表达式中的一个值：可以读取可信决策输入，也可以直接携带字面量。
type ValueRef struct {
	Source ValueSource `json:"source"`
	Path   string      `json:"path,omitempty"`
	Type   ValueType   `json:"type"`
	Value  any         `json:"value,omitempty"`
}
type Comparison struct {
	Left  ValueRef           `json:"left"`
	Op    ComparisonOperator `json:"op"`
	Right *ValueRef          `json:"right,omitempty"`
}

// Condition 是条件语法树节点；All、Any、Not、Comparison 必须且只能设置一个。
type Condition struct {
	All        []Condition `json:"all,omitempty"`
	Any        []Condition `json:"any,omitempty"`
	Not        *Condition  `json:"not,omitempty"`
	Comparison *Comparison `json:"comparison,omitempty"`
}

// UnmarshalJSON 同时接受标准 comparison 结构和设计文档中的紧凑叶子结构。
func (c *Condition) UnmarshalJSON(data []byte) error {
	var raw struct {
		All        []Condition        `json:"all"`
		Any        []Condition        `json:"any"`
		Not        *Condition         `json:"not"`
		Comparison *Comparison        `json:"comparison"`
		Left       *ValueRef          `json:"left"`
		Op         ComparisonOperator `json:"op"`
		Right      *ValueRef          `json:"right"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.All, c.Any, c.Not, c.Comparison = raw.All, raw.Any, raw.Not, raw.Comparison
	if c.Comparison == nil && raw.Left != nil {
		c.Comparison = &Comparison{Left: *raw.Left, Op: raw.Op, Right: raw.Right}
	}
	return nil
}

// Snapshot 是发布后不可变的策略快照，也是评估器唯一接收的策略格式。
type Snapshot struct {
	PolicyID          uuid.UUID         `json:"policy_id"`
	Version           int               `json:"version"`
	AuthorizationType AuthorizationType `json:"authorization_type"`
	ServiceResource   string            `json:"service_resource"`
	Effect            Effect            `json:"effect"`
	Priority          int               `json:"priority"`
	RoleIDs           []string          `json:"role_ids"`
	EndpointIDs       []uuid.UUID       `json:"endpoint_ids"`
	Condition         *Condition        `json:"condition,omitempty"`
}

// Input 是业务服务提交给 PDP 的可信事实。attributes 和 context 只有被条件引用时才需要提供。
type Input struct {
	RequestID         string            `json:"request_id"`
	AuthorizationType AuthorizationType `json:"authorization_type"`
	ServiceResource   string            `json:"service_resource"`
	TenantID          string            `json:"tenant_id"`
	Subject           Subject           `json:"subject"`
	Action            Action            `json:"action"`
	Resource          Resource          `json:"resource"`
	Context           map[string]any    `json:"context,omitempty"`
}
type Subject struct {
	ID string `json:"id"`
	// RoleIDs 由鉴权中心根据已验证的 subject 查询，禁止客户端直接传入。
	RoleIDs    []string       `json:"-"`
	Attributes map[string]any `json:"attributes,omitempty"`
}
type Action struct {
	Kind   ActionKind `json:"kind"`
	Method string     `json:"method,omitempty"`
}
type Resource struct {
	Kind       TargetKind     `json:"kind"`
	EndpointID uuid.UUID      `json:"endpoint_id,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
}
type Result struct {
	Decision         Decision    `json:"decision"`
	ReasonCode       ReasonCode  `json:"reason_code"`
	MatchedPolicyIDs []uuid.UUID `json:"matched_policy_ids"`
	SnapshotVersion  int         `json:"snapshot_version"`
}
