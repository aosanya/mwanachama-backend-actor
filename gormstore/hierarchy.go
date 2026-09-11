package gormstore

import (
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HierarchyRow is the GORM row for a [models.Hierarchy].
type HierarchyRow struct {
	ID string `gorm:"primaryKey"`
	// Code is the stable, human-readable "H-<n>" identifier minted once by
	// CreateHierarchy via NextCode — see codesequence.go. Never updated
	// after insert; RenameHierarchy does not touch it.
	Code string `gorm:"uniqueIndex"`
	Name string
}

// BeforeCreate mints an id via uuid.NewString() when the caller left one
// unset — same convention as GroupRow/RoleKindRow.
func (r *HierarchyRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// HierarchyToRow converts a domain Hierarchy to its row shape.
func HierarchyToRow(h models.Hierarchy) HierarchyRow {
	return HierarchyRow{ID: h.ID, Code: h.Code, Name: h.Name}
}

// HierarchyFromRow converts a row back to the domain Hierarchy.
func HierarchyFromRow(r HierarchyRow) models.Hierarchy {
	return models.Hierarchy{ID: r.ID, Code: r.Code, Name: r.Name}
}

// LevelRow is the GORM row for a [models.Level].
//
// HierarchyID stays a plain indexed string column with NO foreign key —
// same posture as GroupRow.ParentID (see that row's doc): AutoMigrate is
// scoped per instance via db.Table(...), and GORM resolves an association's
// target table from the row struct's default name rather than that runtime
// override, so a declared FK would silently point at the wrong table under
// the multi-instance scheme.
type LevelRow struct {
	ID string `gorm:"primaryKey"`
	// Code is the stable, human-readable "L-<n>" identifier minted once by
	// CreateLevel via NextCode — see codesequence.go. Never updated after
	// insert; RenameLevel does not touch it.
	Code            string `gorm:"uniqueIndex"`
	HierarchyID     string `gorm:"index"`
	Name            string
	Depth           int
	IsDefaultAnchor bool
}

// BeforeCreate mints an id via uuid.NewString() when the caller left one
// unset.
func (r *LevelRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// LevelToRow converts a domain Level to its row shape.
func LevelToRow(l models.Level) LevelRow {
	return LevelRow{
		ID:              l.ID,
		Code:            l.Code,
		HierarchyID:     l.HierarchyID,
		Name:            l.Name,
		Depth:           l.Depth,
		IsDefaultAnchor: l.IsDefaultAnchor,
	}
}

// LevelFromRow converts a row back to the domain Level.
func LevelFromRow(r LevelRow) models.Level {
	return models.Level{
		ID:              r.ID,
		Code:            r.Code,
		HierarchyID:     r.HierarchyID,
		Name:            r.Name,
		Depth:           r.Depth,
		IsDefaultAnchor: r.IsDefaultAnchor,
	}
}
