package model

// UserRole is the persistence record for a subject-to-role assignment. The
// application namespace is resolved through the referenced role, so one
// subject can safely have roles in multiple applications without duplicating
// application data in this join table.
type UserRole struct {
	BaseFields
	Subject string
	RoleID  string
}
