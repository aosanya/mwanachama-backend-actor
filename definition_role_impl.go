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
	defs := make(map[string]DefinitionRole, len(roles))
	for _, r := range roles {
		if r.ID == "" {
			return DefinitionRoleSync{}, ErrDefinitionRoleIDRequired
		}
		defs[r.ID] = r
	}
	out := DefinitionRoleSync{Renamed: []RoleKind{}, Retired: []RoleKind{}}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := m.from(tx, roleRoleKind).Where("definition_role_id <> ''").Order("id")
		kinds, err := specstore.List[RoleKind](m.store, q, roleRoleKind)
		if err != nil {
			return err
		}
		for _, k := range kinds {
			def, kept := defs[k.DefinitionRoleID]
			if !kept {
				if k.RetiredAt != "" {
					continue
				}
				retired, err := m.retireDroppedKind(tx, k, actorID)
				if err != nil {
					return err
				}
				out.Retired = append(out.Retired, retired)
				continue
			}
			if def.Staff != k.Staff {
				k.Staff = def.Staff
				if err := m.from(tx, roleRoleKind).Where("id = ?", k.ID).
					Updates(map[string]any{"staff": def.Staff}).Error; err != nil {
					return err
				}
			}
			if def.Name != "" && def.Name != k.Name {
				k.Name = def.Name
				if err := m.from(tx, roleRoleKind).Where("id = ?", k.ID).
					Updates(map[string]any{"name": def.Name}).Error; err != nil {
					return err
				}
				out.Renamed = append(out.Renamed, withCapabilities(k))
			}
		}
		return nil
	})
	if err != nil {
		return DefinitionRoleSync{}, fmt.Errorf("SyncDefinitionRoles: %w", err)
	}
	return out, nil
}

func (m *userManager) retireDroppedKind(tx *gorm.DB, k RoleKind, actorID string) (RoleKind, error) {
	now := NowRFC3339()
	if err := m.from(tx, roleActorRoleAssignment).
		Where("kind_id = ? AND active", k.ID).
		Updates(map[string]any{"active": false, "ended_at": now, "ended_reason": string(EndedByRoleRemoved)}).Error; err != nil {
		return RoleKind{}, err
	}
	k.RetiredAt = now
	k.RetiredBy = actorID
	if err := m.from(tx, roleRoleKind).Where("id = ?", k.ID).
		Updates(map[string]any{"retired_at": k.RetiredAt, "retired_by": k.RetiredBy}).Error; err != nil {
		return RoleKind{}, err
	}
	return withCapabilities(k), nil
}
