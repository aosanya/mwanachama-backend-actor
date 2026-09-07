package models

// RoleKind is an editable position with a capability set (e.g.
// "coordinator"). Ported from mwanachama-backend-api-gateway's
// internal/domain/role.Kind field-for-field, renamed to avoid a bare `Kind`
// sitting in a models package that already has Actor/Group/
// ActorGroupAssignment — see this repo's CLAUDE.md for the full porting note.
//
// Description and IsDelegate are informational/structural, not enforced by
// this package — Description is free text a caller renders, IsDelegate marks
// a kind that represents delegation to the next group up rather than a
// capability-bearing seat. LevelID scopes a kind to one hierarchy level
// (empty means every level) — a plain string, not FK'd to the Level table
// (see gormstore/group.go's GroupRow doc for why columns that reference
// another row in this package stay plain/indexed rather than a declared
// GORM association). DeleteLevel checks this column before removing a rung
// — see hierarchy_impl.go.
//
// RetiredAt and RetiredBy are the kind's ending (DEV-1133). They are set
// together and cleared together, and RetiredBy is the one field on this type
// this package's own UserManager still resolves from an actor id — see
// RetireRoleKind.
type RoleKind struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Description  string   `json:"description,omitempty"`
	IsDelegate   bool     `json:"is_delegate,omitempty"`
	LevelID      string   `json:"level_id,omitempty"`
	// RetiredAt/RetiredBy carry `omitempty` so a **live** kind serialises
	// exactly as it did before either was ever set — no client parsing a
	// role kind sees a new field until one is actually retired.
	RetiredAt string `json:"retired_at,omitempty"`
	RetiredBy string `json:"retired_by,omitempty"`
}

// Retired reports whether the kind has been retired.
func (k RoleKind) Retired() bool { return k.RetiredAt != "" }

// EndReason records which of the four acts ended an ActorRoleAssignment —
// mirrors mwanachama-backend-api-gateway's internal/domain/role.EndReason.
// Revoke and StepDown deactivate the same row for different acts, open to
// different callers; EndOnEviction/EndOnDeparture are the two further ways a
// seat falls when the membership under it ends.
type EndReason string

const (
	EndedByRevocation EndReason = "revoked"
	EndedByResignation EndReason = "resigned"
	EndedByEviction    EndReason = "evicted"
	EndedByDeparture   EndReason = "deregistered"
)
