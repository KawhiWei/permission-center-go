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
		{PolicyID: allowID, Version: 2, ServiceResource: "orders", Effect: model.EffectAllow, ScopeLevel: model.ScopeAPI, RoleIDs: []string{"editor"}, EndpointIDs: []uuid.UUID{endpointID}},
		{PolicyID: denyID, Version: 3, ServiceResource: "orders", Effect: model.EffectDeny, ScopeLevel: model.ScopeAPI, RoleIDs: []string{"editor"}, EndpointIDs: []uuid.UUID{endpointID}},
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
	result, err := policyeval.Evaluate(apiInput(endpointID, model.Subject{ID: "subject-a", RoleIDs: []string{"sales"}, Attributes: map[string]any{"department": "sales"}}), []model.Snapshot{{PolicyID: uuid.New(), Version: 1, ServiceResource: "orders", Effect: model.EffectAllow, ScopeLevel: model.ScopeAPI, RoleIDs: []string{"sales"}, EndpointIDs: []uuid.UUID{endpointID}, Condition: &condition}})
	if err != nil || result.Decision != model.DecisionAllow {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func apiInput(endpointID uuid.UUID, subject model.Subject) model.Input {
	return model.Input{ServiceResource: "orders", TenantID: "tenant-a", Subject: subject, Action: model.Action{Kind: model.ActionHTTP, Method: "GET"}, Resource: model.Resource{Kind: model.TargetAPIEndpoint, EndpointID: endpointID}}
}
