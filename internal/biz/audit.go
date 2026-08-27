package biz

import (
	"context"
	"strings"
	"time"
)

// BaseFields is the common audit projection shared by all permission records.
// Audit identities are supplied by the authenticated request context, never by request JSON.
type BaseFields struct {
	CreatedByID   string    `json:"created_by_id"`
	CreatedByName string    `json:"created_by_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedByID   string    `json:"updated_by_id"`
	UpdatedByName string    `json:"updated_by_name"`
	UpdatedAt     time.Time `json:"updated_at"`
	IsDeleted     bool      `json:"is_deleted"`
}

type AuditActor struct {
	ID   string
	Name string
}

type auditActorContextKey struct{}

func WithAuditActor(ctx context.Context, actor AuditActor) context.Context {
	actor.ID = strings.TrimSpace(actor.ID)
	actor.Name = strings.TrimSpace(actor.Name)
	if actor.ID == "" {
		actor.ID = "system"
	}
	if actor.Name == "" {
		actor.Name = actor.ID
	}
	return context.WithValue(ctx, auditActorContextKey{}, actor)
}

func AuditActorFromContext(ctx context.Context) AuditActor {
	if ctx != nil {
		if actor, ok := ctx.Value(auditActorContextKey{}).(AuditActor); ok && strings.TrimSpace(actor.ID) != "" {
			actor.ID = strings.TrimSpace(actor.ID)
			actor.Name = strings.TrimSpace(actor.Name)
			if actor.Name == "" {
				actor.Name = actor.ID
			}
			return actor
		}
	}
	return AuditActor{ID: "system", Name: "system"}
}

func NewBaseFields(actor AuditActor) BaseFields {
	if strings.TrimSpace(actor.ID) == "" {
		actor = AuditActor{ID: "system", Name: "system"}
	}
	if strings.TrimSpace(actor.Name) == "" {
		actor.Name = actor.ID
	}
	return BaseFields{
		CreatedByID:   actor.ID,
		CreatedByName: actor.Name,
		UpdatedByID:   actor.ID,
		UpdatedByName: actor.Name,
	}
}
