package model

// Role is scoped to exactly one application.
type Role struct {
	BaseFields
	ID          string
	Application string
	Code        string
	Name        string
	Description string
	Enabled     bool
}
