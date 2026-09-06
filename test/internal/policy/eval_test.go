package policy_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	policyeval "github.com/luck/permission-center-go/internal/policy/eval"
	"github.com/luck/permission-center-go/internal/policy/model"
	"github.com/luck/permission-center-go/internal/policy/validate"
)

func TestDenyOverridesAllow(t *testing.T) {
	endpointID, allowID, denyID := uuid.New(), uuid.New(), uuid.New()
	input := apiInput(endpointID, model.Subject{ID: "subject-a", RoleIDs: []string{"editor"}})
	result, err := policyeval.Evaluate(input, []model.Snapshot{
		{PolicyID: allowID, AuthorizationType: model.AuthorizationTypeAPI, Version: 2, ServiceResource: "orders", Effect: model.EffectAllow, RoleIDs: []string{"editor"}, EndpointIDs: []uuid.UUID{endpointID}},
		{PolicyID: denyID, AuthorizationType: model.AuthorizationTypeAPI, Version: 3, ServiceResource: "orders", Effect: model.EffectDeny, RoleIDs: []string{"editor"}, EndpointIDs: []uuid.UUID{endpointID}},
	})
	if err != nil || result.Decision != model.DecisionDeny || result.ReasonCode != model.ReasonExplicitDeny {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestNoMatchingAllowDefaultsToDeny(t *testing.T) {
	result, err := policyeval.Evaluate(apiInput(uuid.New(), model.Subject{ID: "subject-a", RoleIDs: []string{"viewer"}}), nil)
	if err != nil || result.Decision != model.DecisionDeny || result.ReasonCode != model.ReasonDefaultDeny {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestDesignDocumentCompactConditionIsAccepted(t *testing.T) {
	var condition model.Condition
	if err := json.Unmarshal([]byte(`{"left":{"source":"subject","path":"department","type":"string"},"op":"eq","right":{"source":"literal","type":"string","value":"sales"}}`), &condition); err != nil {
		t.Fatal(err)
	}
	if err := validate.Condition(&condition); err != nil {
		t.Fatalf("condition validation: %v", err)
	}
	endpointID := uuid.New()
	result, err := policyeval.Evaluate(apiInput(endpointID, model.Subject{ID: "subject-a", RoleIDs: []string{"sales"}, Attributes: map[string]any{"department": "sales"}}), []model.Snapshot{{PolicyID: uuid.New(), AuthorizationType: model.AuthorizationTypeAPI, Version: 1, ServiceResource: "orders", Effect: model.EffectAllow, RoleIDs: []string{"sales"}, EndpointIDs: []uuid.UUID{endpointID}, Condition: &condition}})
	if err != nil || result.Decision != model.DecisionAllow {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestMissingAuthorizationTypeIsInvalid(t *testing.T) {
	input := apiInput(uuid.New(), model.Subject{ID: "subject-a", RoleIDs: []string{"viewer"}})
	input.AuthorizationType = ""
	result, err := policyeval.Evaluate(input, nil)
	if err == nil || result.Decision != model.DecisionDeny || result.ReasonCode != model.ReasonInvalidInput {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUnknownAuthorizationTypeIsInvalid(t *testing.T) {
	input := apiInput(uuid.New(), model.Subject{ID: "subject-a", RoleIDs: []string{"viewer"}})
	input.AuthorizationType = "unknown"
	result, err := policyeval.Evaluate(input, nil)
	if err == nil || result.Decision != model.DecisionDeny || result.ReasonCode != model.ReasonInvalidInput {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestMissingAttributeDoesNotMatchComparison(t *testing.T) {
	endpointID := uuid.New()
	condition := &model.Condition{Comparison: &model.Comparison{
		Left:  model.ValueRef{Source: model.SourceSubject, Path: "department", Type: model.TypeString},
		Op:    model.OpNeq,
		Right: &model.ValueRef{Source: model.SourceLiteral, Type: model.TypeString, Value: "finance"},
	}}
	result, err := policyeval.Evaluate(apiInput(endpointID, model.Subject{ID: "subject-a", RoleIDs: []string{"viewer"}}), []model.Snapshot{{PolicyID: uuid.New(), AuthorizationType: model.AuthorizationTypeAPI, Version: 1, ServiceResource: "orders", Effect: model.EffectAllow, RoleIDs: []string{"viewer"}, EndpointIDs: []uuid.UUID{endpointID}, Condition: condition}})
	if err != nil || result.Decision != model.DecisionDeny || result.ReasonCode != model.ReasonDefaultDeny {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestDataAuthorizationOnlyMatchesDataPolicies(t *testing.T) {
	endpointID := uuid.New()
	input := apiInput(endpointID, model.Subject{ID: "subject-a", RoleIDs: []string{"viewer"}})
	input.AuthorizationType = model.AuthorizationTypeData
	result, err := policyeval.Evaluate(input, []model.Snapshot{
		{PolicyID: uuid.New(), AuthorizationType: model.AuthorizationTypeAPI, Version: 1, ServiceResource: "orders", Effect: model.EffectAllow, RoleIDs: []string{"viewer"}, EndpointIDs: []uuid.UUID{endpointID}},
		{PolicyID: uuid.New(), AuthorizationType: model.AuthorizationTypeData, Version: 1, ServiceResource: "orders", Effect: model.EffectAllow, RoleIDs: []string{"viewer"}, EndpointIDs: []uuid.UUID{endpointID}},
	})
	if err != nil || result.Decision != model.DecisionAllow || len(result.MatchedPolicyIDs) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func apiInput(endpointID uuid.UUID, subject model.Subject) model.Input {
	return model.Input{AuthorizationType: model.AuthorizationTypeAPI, ServiceResource: "orders", TenantID: "tenant-a", Subject: subject, Action: model.Action{Kind: model.ActionHTTP, Method: "GET"}, Resource: model.Resource{Kind: model.TargetAPIEndpoint, EndpointID: endpointID}}
}
