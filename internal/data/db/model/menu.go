package model

import "github.com/google/uuid"

type MenuType string

const (
	MenuTypeMenu   MenuType = "menu"
	MenuTypeButton MenuType = "button"
)

// Menu is a node in an application's menu/button tree. A nil ParentID
// denotes a root menu; button roots are rejected by the database constraint.
type Menu struct {
	BaseFields
	ID              uuid.UUID
	ServiceResource string
	Application     string
	ParentID        *uuid.UUID
	Code            string
	Name            string
	Description     string
	Type            MenuType
	Path            string
	Component       string
	APIPath         string
	HTTPMethod      string
	Icon            string
	Sort            int
	Metadata        []byte
	Enabled         bool
}
