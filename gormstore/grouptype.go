package gormstore

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

// GroupTypeRow is the GORM row for a [models.GroupType].
type GroupTypeRow struct {
	ID string `gorm:"primaryKey"`
	// Code is the stable, human-readable "GT-<n>" identifier minted once by
	// CreateGroupType via NextCode — see codesequence.go. Never updated
	// after insert.
	Code        string `gorm:"uniqueIndex"`
	HierarchyID string `gorm:"index"`
	Name        string
	Singular    string
	Plural      string
	LevelID     string `gorm:"index"`
}

// BeforeCreate mints an id via uuid.NewString() when the caller left one
// unset.
func (r *GroupTypeRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// GroupTypeToRow converts a domain GroupType to its row shape.
func GroupTypeToRow(g models.GroupType) GroupTypeRow {
	return GroupTypeRow{
		ID:          g.ID,
		Code:        g.Code,
		HierarchyID: g.HierarchyID,
		Name:        g.Name,
		Singular:    g.Singular,
		Plural:      g.Plural,
		LevelID:     g.LevelID,
	}
}

// GroupTypeFromRow converts a row back to the domain GroupType.
func GroupTypeFromRow(r GroupTypeRow) models.GroupType {
	return models.GroupType{
		ID:          r.ID,
		Code:        r.Code,
		HierarchyID: r.HierarchyID,
		Name:        r.Name,
		Singular:    r.Singular,
		Plural:      r.Plural,
		LevelID:     r.LevelID,
	}
}
