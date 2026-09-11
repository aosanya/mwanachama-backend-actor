package models

// Group is a node in the org structure — the generic type the gateway's
// domain calls "Chapter" (DSN-1698 decision 3) and presents under an
// org-configurable label. ParentID is empty for the root.
//
// HierarchyID, LevelID, and AnchorLevelOverrideID name rows that live in the
// gateway's OWN Postgres tables (hierarchy, chapter_level) — see the
// DSN-1699 gap 1 default recorded in the gormstore package's GroupRow doc.
// This package carries them as opaque strings and does not validate them
// against anything; the gateway adapter is responsible for whatever
// validation it still wants to do against its own hierarchy/chapter_level
// tables.
type Group struct {
	ID string `json:"id"`
	// Code is a stable, human-readable identifier (e.g. "G-1"), assigned
	// once at creation and never changed afterward — see gormstore's
	// NextCode. Unlike Name, it is never editable (GroupEdit carries no
	// Code field).
	Code                  string `json:"code"`
	HierarchyID           string `json:"hierarchy_id"`
	LevelID               string `json:"level_id"`
	ParentID              string `json:"parent_id,omitempty"`
	Name                  string `json:"name"`
	Discoverable          bool   `json:"discoverable"`
	AnchorLevelOverrideID string `json:"anchor_level_override,omitempty"`
	NodeType              string `json:"node_type,omitempty"`
	CreatedAt             string `json:"created_at"`
	// LastUpdated is stamped on every write — see gormstore's
	// GroupRow.UpdatedAt, which this mirrors.
	LastUpdated string `json:"last_updated"`
	// Deleted marks a soft-deleted group. Not `omitempty` — see Actor's
	// identical field for why. No delete method sets it yet; ListGroups
	// filters it out regardless, so the column and the filter land together.
	Deleted bool `json:"deleted"`

	// Attributes is an open prop:value map for organization-declared group
	// properties, validated against DefaultGroupProperties the same way
	// Actor.Attributes is validated against DefaultActorProperties — see
	// property.go. Added 2026-09-04, alongside Actor's catalog.
	// `omitempty`: an absent map and an empty one mean the same thing.
	Attributes map[string]any `json:"attributes,omitempty"`
}

// GroupEdit is the whole of what EditGroup may write. Mirrors the gateway's
// chapter.Edit. Empty clears — every field is written on every call.
type GroupEdit struct {
	Name                  string
	NodeType              string
	AnchorLevelOverrideID string
}

// DefaultGroupProperties is the built-in property catalog validated against
// Group.Attributes on every CreateGroup. Empty for now — no built-in group
// property has been decided yet — but the mechanism mirrors Actor's: an
// organization-declared property added here immediately gets
// Required/Unique/Range validation for free.
func DefaultGroupProperties() []Property {
	return nil
}
