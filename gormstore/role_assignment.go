package gormstore

import (
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ActorRoleAssignmentRow is the GORM row for a [models.ActorRoleAssignment].
//
// ID is a caller-visible minted string (unlike ActorGroupAssignmentRow's
// composite (actor_id, group_id) key) because a seat is granted, revoked and
// stepped down from by id — the gateway's own role_assignment.id primary key
// this ports, not a relationship uniqueness constraint.
type ActorRoleAssignmentRow struct {
	ID          string `gorm:"primaryKey"`
	ActorID     string `gorm:"index"`
	GroupID     string `gorm:"index"`
	KindID      string `gorm:"index"`
	Active      bool
	GrantedAt   string
	GrantedBy   string
	EndedAt     string
	EndedReason string
}

// BeforeCreate mints an id via uuid.NewString() when the caller left one
// unset.
func (r *ActorRoleAssignmentRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ActorRoleAssignmentToRow converts a domain assignment to its row shape.
func ActorRoleAssignmentToRow(a models.ActorRoleAssignment) ActorRoleAssignmentRow {
	return ActorRoleAssignmentRow{
		ID:          a.ID,
		ActorID:     a.ActorID,
		GroupID:     a.GroupID,
		KindID:      a.KindID,
		Active:      a.Active,
		GrantedAt:   a.GrantedAt,
		GrantedBy:   a.GrantedBy,
		EndedAt:     a.EndedAt,
		EndedReason: string(a.EndedReason),
	}
}

// ActorRoleAssignmentFromRow converts a row back to the domain assignment.
func ActorRoleAssignmentFromRow(r ActorRoleAssignmentRow) models.ActorRoleAssignment {
	return models.ActorRoleAssignment{
		ID:          r.ID,
		ActorID:     r.ActorID,
		GroupID:     r.GroupID,
		KindID:      r.KindID,
		Active:      r.Active,
		GrantedAt:   r.GrantedAt,
		GrantedBy:   r.GrantedBy,
		EndedAt:     r.EndedAt,
		EndedReason: models.EndReason(r.EndedReason),
	}
}
