package gormstore

import (
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// GroupRow is the GORM row for a [models.Group].
//
// HierarchyID, LevelID, and AnchorLevelOverrideID stay plain opaque string
// columns with NO foreign key — those rows live in
// mwanachama-backend-api-gateway's own hierarchy/chapter_level tables (the
// DSN-1699 gap 1 default), which this package still does not depend on. That
// has nothing to do with graphs vs. relational storage, so it carries over
// unchanged.
//
// ParentID is a nullable *string (NULL for the root), not the domain
// Group's plain "" — GroupToRow/GroupFromRow map "" <-> nil at the boundary.
// It is indexed but deliberately NOT declared as a GORM association/foreign
// key: Migrate scopes AutoMigrate per instance via db.Table(...) (see
// tables.go), but GORM resolves an association's target table from the row
// struct's default name, not that runtime override — a self-referencing FK
// here would silently point at the wrong table under the multi-instance
// scheme. Plain indexed column instead; MoveGroup's cycle/subtree check
// stays a Go walk over ListGroups in the root package's group_impl.go,
// unaffected either way.
type GroupRow struct {
	ID string `gorm:"primaryKey"`
	// Code is the stable, human-readable "G-<n>" identifier minted once by
	// CreateGroup via NextCode — see codesequence.go. Never updated after
	// insert; EditGroup does not touch it.
	Code                  string `gorm:"uniqueIndex"`
	HierarchyID           string
	LevelID               string
	ParentID              *string `gorm:"index"`
	Name                  string
	Discoverable          bool
	AnchorLevelOverrideID string `gorm:"column:anchor_level_override"`
	NodeType              string
	// Attributes is the organization-declared group property blob, stored
	// as native JSONB the same way ActorRow's Attributes is — see
	// models.Group's doc. Added 2026-09-04.
	Attributes datatypes.JSONMap
	CreatedAt  string
	UpdatedAt  string
	Deleted    bool
}

func (r *GroupRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// GroupToRow converts a domain Group to its row shape, stamping UpdatedAt.
func GroupToRow(g models.Group) GroupRow {
	var attrs datatypes.JSONMap
	if len(g.Attributes) > 0 {
		attrs = datatypes.JSONMap(g.Attributes)
	}
	return GroupRow{
		ID:                    g.ID,
		Code:                  g.Code,
		Name:                  g.Name,
		HierarchyID:           g.HierarchyID,
		LevelID:               g.LevelID,
		ParentID:              StringToNullable(g.ParentID),
		Discoverable:          g.Discoverable,
		AnchorLevelOverrideID: g.AnchorLevelOverrideID,
		NodeType:              g.NodeType,
		Attributes:            attrs,
		CreatedAt:             g.CreatedAt,
		UpdatedAt:             models.NowRFC3339(),
		Deleted:               g.Deleted,
	}
}

// GroupFromRow converts a row back to the domain Group.
func GroupFromRow(r GroupRow) models.Group {
	g := models.Group{
		ID:                    r.ID,
		Code:                  r.Code,
		Name:                  r.Name,
		HierarchyID:           r.HierarchyID,
		LevelID:               r.LevelID,
		ParentID:              nullableToString(r.ParentID),
		Discoverable:          r.Discoverable,
		AnchorLevelOverrideID: r.AnchorLevelOverrideID,
		NodeType:              r.NodeType,
		CreatedAt:             r.CreatedAt,
		LastUpdated:           r.UpdatedAt,
		Deleted:               r.Deleted,
	}
	if len(r.Attributes) > 0 {
		g.Attributes = map[string]any(r.Attributes)
	}
	return g
}

// StringToNullable maps the domain Group's "" (root) to a nil *string, the
// nullable ParentID column's spelling of "no parent". Exported for
// group_impl.go's MoveGroup, which writes parent_id directly rather than
// through GroupToRow.
func StringToNullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullableToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
