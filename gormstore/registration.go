package gormstore

import (
	"github.com/aosanya/mwanachama-backend-actor/models"
	"gorm.io/datatypes"
)

// ActorGroupAssignmentRow is the GORM row for the registration relationship
// between an Actor and a Group (see [models.ActorGroupAssignment]). The
// composite primary key (actor_id, group_id) gives natural (actor, group)
// uniqueness for free — unlike entitygraph, which had no UpdateRelationship
// and forced the root package's AssignGroup into a delete-then-recreate
// workaround.
//
// No IsHome/JoinedAt columns, 2026-09-04 — see models.ActorGroupAssignment's
// doc for where that fact lives now (CreatedAt, or Attributes for anything
// more specific).
type ActorGroupAssignmentRow struct {
	ActorID   string `gorm:"primaryKey"`
	GroupID   string `gorm:"primaryKey"`
	CreatedAt string
	UpdatedAt string
	Deleted   bool
	// Attributes is the organization-declared assignment property blob,
	// stored as native JSONB the same way ActorRow's and GroupRow's
	// Attributes are. Added 2026-09-04.
	Attributes datatypes.JSONMap
}

// ActorGroupAssignmentToRow converts a domain ActorGroupAssignment to its
// row shape.
func ActorGroupAssignmentToRow(r models.ActorGroupAssignment) ActorGroupAssignmentRow {
	var attrs datatypes.JSONMap
	if len(r.Attributes) > 0 {
		attrs = datatypes.JSONMap(r.Attributes)
	}
	return ActorGroupAssignmentRow{
		ActorID:    r.ActorID,
		GroupID:    r.GroupID,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.LastUpdated,
		Deleted:    r.Deleted,
		Attributes: attrs,
	}
}

// ActorGroupAssignmentFromRow converts a row back to the domain
// ActorGroupAssignment.
func ActorGroupAssignmentFromRow(r ActorGroupAssignmentRow) models.ActorGroupAssignment {
	a := models.ActorGroupAssignment{
		ActorID:     r.ActorID,
		GroupID:     r.GroupID,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
	if len(r.Attributes) > 0 {
		a.Attributes = map[string]any(r.Attributes)
	}
	return a
}
