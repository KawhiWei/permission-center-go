package model

import "github.com/google/uuid"

type ResourceType string

const (
	ResourceTypeMenu   ResourceType = "menu"
	ResourceTypeButton ResourceType = "button"
)

// Resource is a node in an application's menu/button tree. A nil ParentID
// denotes a root menu; button roots are rejected by the database constraint.
type Resource struct {
	BaseFields
	ID          uuid.UUID
	Application string
	ParentID    *uuid.UUID
	Code        string
	Name        string
	Description string
	Type        ResourceType
	Path        string
	Component   string
	APIPath     string
	HTTPMethod  string
	Icon        string
	Sort        int
	Metadata    []byte
	Enabled     bool
}
