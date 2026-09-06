// Package eval 实现不访问数据库和网络的 PDP 评估器，策略合并规则为拒绝优先。
package eval

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/luck/permission-center-go/internal/policy/model"
)

// Evaluate 根据必填的 authorization_type 选择对应策略，未知或缺失类型直接拒绝。
func Evaluate(input model.Input, snapshots []model.Snapshot) (model.Result, error) {
	strategy, ok := authorizationStrategies[input.AuthorizationType]
	if !ok {
		return model.Result{Decision: model.DecisionDeny, ReasonCode: model.ReasonInvalidInput, MatchedPolicyIDs: []uuid.UUID{}}, fmt.Errorf("authorization_type must be api or data")
	}
	return strategy.Evaluate(input, snapshots)
}

type authorizationStrategy interface {
	Evaluate(model.Input, []model.Snapshot) (model.Result, error)
}

var authorizationStrategies = map[model.AuthorizationType]authorizationStrategy{
	model.AuthorizationTypeAPI:  apiAuthorizationStrategy{},
	model.AuthorizationTypeData: dataAuthorizationStrategy{},
}

type apiAuthorizationStrategy struct{}

// API 策略只判断调用者是否允许访问指定 API 端点，不要求 attributes。
func (apiAuthorizationStrategy) Evaluate(input model.Input, snapshots []model.Snapshot) (model.Result, error) {
	return evaluateByAuthorizationType(input, snapshots, model.AuthorizationTypeAPI)
}

type dataAuthorizationStrategy struct{}

// 数据策略在同一 API 入口上结合可选 attributes 判断具体业务对象是否允许访问。
func (dataAuthorizationStrategy) Evaluate(input model.Input, snapshots []model.Snapshot) (model.Result, error) {
	return evaluateByAuthorizationType(input, snapshots, model.AuthorizationTypeData)
}

func evaluateByAuthorizationType(input model.Input, snapshots []model.Snapshot, authorizationType model.AuthorizationType) (model.Result, error) {
	if strings.TrimSpace(input.ServiceResource) == "" || strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.Subject.ID) == "" || input.Resource.Kind != model.TargetAPIEndpoint || input.Resource.EndpointID == uuid.Nil || !input.Action.Kind.Valid() {
		return model.Result{Decision: model.DecisionDeny, ReasonCode: model.ReasonInvalidInput, MatchedPolicyIDs: []uuid.UUID{}}, fmt.Errorf("service_resource, tenant_id, subject.id and endpoint_id are required")
	}
	result := model.Result{Decision: model.DecisionDeny, ReasonCode: model.ReasonDefaultDeny, MatchedPolicyIDs: []uuid.UUID{}}
	allow := false
	for _, snapshot := range snapshots {
		if snapshot.AuthorizationType != authorizationType || snapshot.ServiceResource != input.ServiceResource || !hasEndpoint(snapshot.EndpointIDs, input.Resource.EndpointID) || !hasRole(snapshot.RoleIDs, input.Subject.RoleIDs) {
			continue
		}
		matches, err := matchesCondition(snapshot.Condition, input)
		if err != nil {
			return result, err
		}
		if !matches {
			continue
		}
		result.MatchedPolicyIDs = append(result.MatchedPolicyIDs, snapshot.PolicyID)
		if snapshot.Version > result.SnapshotVersion {
			result.SnapshotVersion = snapshot.Version
		}
		// 任意一条 deny 命中即立即拒绝，避免 allow 策略覆盖显式拒绝。
		if snapshot.Effect == model.EffectDeny {
			result.Decision = model.DecisionDeny
			result.ReasonCode = model.ReasonExplicitDeny
			return result, nil
		}
		allow = true
	}
	if allow {
		result.Decision = model.DecisionAllow
		result.ReasonCode = model.ReasonAllowedByPolicy
	}
	return result, nil
}
func hasEndpoint(values []uuid.UUID, id uuid.UUID) bool {
	for _, value := range values {
		if value == id {
			return true
		}
	}
	return false
}
func hasRole(policy, subject []string) bool {
	for _, p := range policy {
		for _, s := range subject {
			if p == s {
				return true
			}
		}
	}
	return false
}
func matchesCondition(condition *model.Condition, input model.Input) (bool, error) {
	if condition == nil {
		return true, nil
	}
	if len(condition.All) > 0 {
		for i := range condition.All {
			ok, e := matchesCondition(&condition.All[i], input)
			if e != nil || !ok {
				return ok, e
			}
		}
		return true, nil
	}
	if len(condition.Any) > 0 {
		for i := range condition.Any {
			ok, e := matchesCondition(&condition.Any[i], input)
			if e != nil {
				return false, e
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	if condition.Not != nil {
		ok, e := matchesCondition(condition.Not, input)
		return !ok, e
	}
	return comparison(*condition.Comparison, input)
}
func comparison(value model.Comparison, input model.Input) (bool, error) {
	left, exists, err := resolve(value.Left, input)
	if err != nil {
		return false, err
	}
	if value.Op == model.OpExists {
		return exists, nil
	}
	if !exists {
		return false, nil
	}
	right, rightExists, err := resolve(*value.Right, input)
	if err != nil {
		return false, err
	}
	if !rightExists {
		return false, nil
	}
	switch value.Op {
	case model.OpEq:
		return reflect.DeepEqual(left, right), nil
	case model.OpNeq:
		return !reflect.DeepEqual(left, right), nil
	case model.OpIn:
		return contains(right, left), nil
	case model.OpNotIn:
		return !contains(right, left), nil
	case model.OpContains:
		return contains(left, right), nil
	}
	return false, fmt.Errorf("unsupported operator")
}
func resolve(ref model.ValueRef, input model.Input) (any, bool, error) {
	// literal 直接取策略中的常量，其他 source 只能读取服务端构造的可信输入。
	if ref.Source == model.SourceLiteral {
		return ref.Value, true, nil
	}
	var root map[string]any
	switch ref.Source {
	case model.SourceSubject:
		root = input.Subject.Attributes
		if ref.Path == "id" {
			return input.Subject.ID, true, nil
		}
	case model.SourceResource:
		root = input.Resource.Attributes
	case model.SourceRequest:
		root = map[string]any{"tenant_id": input.TenantID, "method": input.Action.Method}
	case model.SourceContext:
		root = input.Context
	}
	value, ok := root[ref.Path]
	return value, ok, nil
}
func contains(container, value any) bool {
	rv := reflect.ValueOf(container)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array && rv.Kind() != reflect.String {
		return false
	}
	if rv.Kind() == reflect.String {
		v, ok := value.(string)
		return ok && strings.Contains(rv.String(), v)
	}
	for i := 0; i < rv.Len(); i++ {
		if reflect.DeepEqual(rv.Index(i).Interface(), value) {
			return true
		}
	}
	return false
}
