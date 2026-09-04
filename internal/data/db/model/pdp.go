package model

import (
	"github.com/google/uuid"
)

// AuthorizationResource is a registered business resource. It contains only
// metadata used by the PDP; business instances and their attributes remain in
// the owning business service.
type AuthorizationResource struct {
	BaseFields
	ID              uuid.UUID
	ServiceResource string
	Application     string
	Code            string
	Type            string
	Name            string
	Description     string
	Matcher         string
	Enabled         bool
}

// AuthorizationAction is a business operation that can be selected by a
// policy. HTTP methods are deliberately kept on API endpoint registrations,
// rather than being mixed into this model.
type AuthorizationAction struct {
	BaseFields
	ID              uuid.UUID
	ServiceResource string
	Application     string
	Code            string
	Name            string
	Description     string
	Enabled         bool
}

// AuthorizationAPIEndpoint binds a normalized business route to a resource
// and action. It is independent from the UI menu/button tree.
type AuthorizationAPIEndpoint struct {
	BaseFields
	ID              uuid.UUID
	ServiceResource string
	Application     string
	ServiceCode     string
	Method          string
	PathTemplate    string
	ResourceID      uuid.UUID
	ActionID        uuid.UUID
	EnforcementMode string
	Enabled         bool
}

// AuthorizationPolicy is an RBAC-selectable rule. Conditions and obligations
// are intentionally deferred to later PDP phases; selectors support exact
// values and the * wildcard in Phase A.
type AuthorizationPolicy struct {
	BaseFields
	ID              uuid.UUID
	ServiceResource string
	Application     string
	Code            string
	Name            string
	Description     string
	Effect          string
	Priority        int
	ResourceCodes   []string
	ActionCodes     []string
	Enabled         bool
}

// AuthorizationPolicyBinding attaches one policy to a role or a concrete
// external subject. Subject values are bounded to the same 80-character
// contract as user_roles and role IDs.
type AuthorizationPolicyBinding struct {
	BaseFields
	PolicyID     uuid.UUID
	SubjectType  string
	SubjectValue string
	Enabled      bool
}
