package gormstore

import (
	"github.com/aosanya/mwanachama-backend-actor/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RoleKindRow is the GORM row for a [models.RoleKind].
type RoleKindRow struct {
	ID           string `gorm:"primaryKey"`
	Name         string
	Capabilities datatypes.JSON
	Description  string
	IsDelegate   bool
	LevelID      string
	RetiredAt    string
	RetiredBy    string
}

// BeforeCreate mints an id via uuid.NewString() when the caller left one
// unset — same convention as ActorRow/GroupRow.
func (r *RoleKindRow) BeforeCreate(_ *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	return nil
}

// RoleKindToRow converts a domain RoleKind to its row shape. Capabilities
// round-trips through a JSON array rather than datatypes.JSONMap (a map),
// matching the gateway's own jsonb-array column.
func RoleKindToRow(k models.RoleKind) (RoleKindRow, error) {
	caps, err := marshalStrings(k.Capabilities)
	if err != nil {
		return RoleKindRow{}, err
	}
	return RoleKindRow{
		ID:           k.ID,
		Name:         k.Name,
		Capabilities: caps,
		Description:  k.Description,
		IsDelegate:   k.IsDelegate,
		LevelID:      k.LevelID,
		RetiredAt:    k.RetiredAt,
		RetiredBy:    k.RetiredBy,
	}, nil
}

// RoleKindFromRow converts a row back to the domain RoleKind. Capabilities
// is never nil on the way out — DEV-1077's rule, ported: a kind created
// without capabilities must serialise as `[]`, not `null`.
func RoleKindFromRow(r RoleKindRow) (models.RoleKind, error) {
	caps, err := unmarshalStrings(r.Capabilities)
	if err != nil {
		return models.RoleKind{}, err
	}
	if caps == nil {
		caps = []string{}
	}
	return models.RoleKind{
		ID:           r.ID,
		Name:         r.Name,
		Capabilities: caps,
		Description:  r.Description,
		IsDelegate:   r.IsDelegate,
		LevelID:      r.LevelID,
		RetiredAt:    r.RetiredAt,
		RetiredBy:    r.RetiredBy,
	}, nil
}
