// Package gormstore holds every GORM-specific piece of this repo: row
// structs, their conversion to/from the domain types in
// mwanachama-backend-actor/models, and table migration. Nothing outside
// this package (and the root mwanachama-backend-actor package's *_impl.go
// files, which call it) needs to know GORM exists.
package gormstore

import (
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ActorRow is the GORM row for a [models.Actor].
type ActorRow struct {
	ID string `gorm:"primaryKey"`
	// Code is the stable, human-readable "AC-<n>" identifier minted once by
	// CreateActor via NextCode — see codesequence.go. Never updated after
	// insert.
	Code        string `gorm:"uniqueIndex"`
	DisplayName string
	IsAgentic   bool
	// Attributes is the persona blob (DSN-1664's A-Box), stored as native
	// JSONB — a map[string]any round-trip via datatypes.JSONMap, replacing
	// the JSON-encoded-string convention entitygraph's opaque properties
	// required. Phone and email live here too, 2026-09-04 — see
	// models.Actor's doc for why they are no longer their own columns.
	Attributes datatypes.JSONMap
	CreatedAt  string
	UpdatedAt  string
	Deleted    bool
}

// BeforeCreate mints an id via uuid.NewString() when the caller left one
// unset, matching the ID style mwanachama-backend-shared/postgres uses for
// entitygraph entities.
func (r *ActorRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// ActorToRow converts a domain Actor to its row shape, stamping UpdatedAt.
func ActorToRow(a models.Actor) ActorRow {
	var attrs datatypes.JSONMap
	if len(a.Attributes) > 0 {
		attrs = datatypes.JSONMap(a.Attributes)
	}
	return ActorRow{
		ID:          a.ID,
		Code:        a.Code,
		DisplayName: a.DisplayName,
		IsAgentic:   a.IsAgentic,
		Attributes:  attrs,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   models.NowRFC3339(),
		Deleted:     a.Deleted,
	}
}

// ActorFromRow converts a row back to the domain Actor.
func ActorFromRow(r ActorRow) models.Actor {
	a := models.Actor{
		ID:          r.ID,
		Code:        r.Code,
		DisplayName: r.DisplayName,
		IsAgentic:   r.IsAgentic,
		CreatedAt:   r.CreatedAt,
		LastUpdated: r.UpdatedAt,
		Deleted:     r.Deleted,
	}
	if len(r.Attributes) > 0 {
		a.Attributes = map[string]any(r.Attributes)
	}
	return a
}
