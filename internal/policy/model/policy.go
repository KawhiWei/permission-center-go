// Package model defines the serializable, storage-neutral PDP policy contract.
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

type ScopeLevel string

const (
	ScopeAPI   ScopeLevel = "api"
	ScopeRow   ScopeLevel = "row"
	ScopeField ScopeLevel = "field"
)

func (v ScopeLevel) Valid() bool { return v == ScopeAPI || v == ScopeRow || v == ScopeField }

type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

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
	TargetAPIEndpoint  TargetKind = "api_endpoint"
	TargetResourceType TargetKind = "resource_type"
)

func (v TargetKind) Valid() bool { return v == TargetAPIEndpoint || v == TargetResourceType }

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

// ValueRef reads a typed value from decision input, or carries a literal.
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

// Condition is an AST node. Exactly one of All, Any, Not or Comparison must be set.
type Condition struct {
	All        []Condition `json:"all,omitempty"`
	Any        []Condition `json:"any,omitempty"`
	Not        *Condition  `json:"not,omitempty"`
	Comparison *Comparison `json:"comparison,omitempty"`
}

// UnmarshalJSON accepts both the canonical {"comparison": {...}} shape and
// the compact leaf shape from the PDP design document: {"left": ..., "op": ...}.
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

type Obligation struct {
	RowFilter  any         `json:"row_filter,omitempty"`
	FieldRules []FieldRule `json:"field_rules,omitempty"`
}
type FieldRule struct {
	Field  string      `json:"field"`
	Access FieldAccess `json:"access"`
	Mask   string      `json:"mask,omitempty"`
}
type FieldAccess string

const (
	FieldRead      FieldAccess = "read"
	FieldWrite     FieldAccess = "write"
	FieldReadWrite FieldAccess = "read_write"
	FieldMask      FieldAccess = "mask"
	FieldDeny      FieldAccess = "deny"
)

func (v FieldAccess) Valid() bool {
	switch v {
	case FieldRead, FieldWrite, FieldReadWrite, FieldMask, FieldDeny:
		return true
	}
	return false
}

// Snapshot is immutable input to the evaluator. It intentionally contains no database model.
type Snapshot struct {
	PolicyID        uuid.UUID   `json:"policy_id"`
	Version         int         `json:"version"`
	ServiceResource string      `json:"service_resource"`
	Effect          Effect      `json:"effect"`
	ScopeLevel      ScopeLevel  `json:"scope_level"`
	Priority        int         `json:"priority"`
	RoleIDs         []string    `json:"role_ids"`
	EndpointIDs     []uuid.UUID `json:"endpoint_ids"`
	Condition       *Condition  `json:"condition,omitempty"`
	Obligations     Obligation  `json:"obligations"`
}
type Input struct {
	RequestID       string         `json:"request_id"`
	ServiceResource string         `json:"service_resource"`
	TenantID        string         `json:"tenant_id"`
	Subject         Subject        `json:"subject"`
	Action          Action         `json:"action"`
	Resource        Resource       `json:"resource"`
	Context         map[string]any `json:"context,omitempty"`
}
type Subject struct {
	ID         string         `json:"id"`
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
	Obligations      Obligation  `json:"obligations"`
}
