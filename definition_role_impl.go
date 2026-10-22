package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func (m *userManager) RoleKindForDefinitionRole(ctx context.Context, definitionRoleID string) (RoleKind, error) {
	if definitionRoleID == "" {
		return RoleKind{}, ErrDefinitionRoleIDRequired
	}
	var out RoleKind
	q := m.store.Query(ctx, roleRoleKind).Where("definition_role_id = ?", definitionRoleID).Order("id")
	if err := m.store.Take(q, roleRoleKind, &out, ErrRoleKindNotFound); err != nil {
		if errors.Is(err, ErrRoleKindNotFound) {
			return RoleKind{}, ErrRoleKindNotFound
		}
		return RoleKind{}, fmt.Errorf("RoleKindForDefinitionRole: %w", err)
	}
	return withCapabilities(out), nil
}

func (m *userManager) SyncDefinitionRoles(ctx context.Context, roles []DefinitionRole, actorID string) (DefinitionRoleSync, error) {
	names := make(map[string]string, len(roles))
	for _, r := range roles {
		if r.ID == "" {
			return DefinitionRoleSync{}, ErrDefinitionRoleIDRequired
		}
		names[r.ID] = r.Name
	}
	out := DefinitionRoleSync{Renamed: []RoleKind{}, Retired: []RoleKind{}}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Table(m.table(roleRoleKind)).Where("definition_role_id <> ''").Order("id")
		kinds, err := specstore.List[RoleKind](m.store, q, roleRoleKind)
		if err != nil {
			return err
		}
		for _, k := range kinds {
			name, kept := names[k.DefinitionRoleID]
			switch {
			case kept && name != "" && name != k.Name:
				k.Name = name
				if err := tx.Table(m.table(roleRoleKind)).Where("id = ?", k.ID).
					Updates(map[string]any{"name": name}).Error; err != nil {
					return err
				}
				out.Renamed = append(out.Renamed, withCapabilities(k))
			case !kept && k.RetiredAt == "":
				k.RetiredAt = NowRFC3339()
				k.RetiredBy = actorID
				if err := tx.Table(m.table(roleRoleKind)).Where("id = ?", k.ID).
					Updates(map[string]any{"retired_at": k.RetiredAt, "retired_by": k.RetiredBy}).Error; err != nil {
					return err
				}
				out.Retired = append(out.Retired, withCapabilities(k))
			}
		}
		return nil
	})
	if err != nil {
		return DefinitionRoleSync{}, fmt.Errorf("SyncDefinitionRoles: %w", err)
	}
	return out, nil
}
