package model

import "github.com/google/uuid"

// RoleResource records one role-to-resource grant.
type RoleResource struct {
	BaseFields
	RoleID     string
	ResourceID uuid.UUID
}
