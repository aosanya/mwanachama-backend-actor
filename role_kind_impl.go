package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func withCapabilities(k RoleKind) RoleKind {
	if k.Capabilities == nil {
		k.Capabilities = []string{}
	}
	return k
}

func (m *userManager) CreateRoleKind(ctx context.Context, k RoleKind) (RoleKind, error) {
	k.ID = newSpecID()
	k.RetiredAt, k.RetiredBy = "", ""

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := nextCode(ctx, tx, m.table(roleCodeSequence), "role_kind", "RK")
		if err != nil {
			return err
		}
		k.Code = code
		row, err := encode(m.object(roleRoleKind), k)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Table(m.table(roleRoleKind)).Create(row).Error; err != nil {
			return classifyDuplicateID(err)
		}
		return nil
	})
	if err != nil {
		return RoleKind{}, fmt.Errorf("CreateRoleKind: %w", classifyConflict(err))
	}
	return withCapabilities(k), nil
}

func (m *userManager) ListRoleKinds(ctx context.Context) ([]RoleKind, error) {
	q := m.store.Query(ctx, roleRoleKind).Order("id")
	rows, err := specstore.List[RoleKind](m.store, q, roleRoleKind)
	if err != nil {
		return nil, fmt.Errorf("ListRoleKinds: %w", err)
	}
	out := make([]RoleKind, 0, len(rows))
	for _, k := range rows {
		out = append(out, withCapabilities(k))
	}
	return out, nil
}

func (m *userManager) GetRoleKind(ctx context.Context, id string) (RoleKind, error) {
	var out RoleKind
	q := m.store.Query(ctx, roleRoleKind).Where("id = ?", id)
	if err := m.store.Take(q, roleRoleKind, &out, ErrRoleKindNotFound); err != nil {
		if errors.Is(err, ErrRoleKindNotFound) {
			return RoleKind{}, ErrRoleKindNotFound
		}
		return RoleKind{}, fmt.Errorf("GetRoleKind: %w", err)
	}
	return withCapabilities(out), nil
}

func (m *userManager) roleKindIn(tx *gorm.DB, id string) (RoleKind, error) {
	var out RoleKind
	q := tx.Table(m.table(roleRoleKind)).Where("id = ?", id)
	if err := m.store.Take(q, roleRoleKind, &out, ErrRoleKindNotFound); err != nil {
		return RoleKind{}, err
	}
	return withCapabilities(out), nil
}

func (m *userManager) RetireRoleKind(ctx context.Context, kindID, actorID string) (RoleKind, error) {
	var out RoleKind
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		k, err := m.roleKindIn(tx, kindID)
		if err != nil {
			return err
		}
		if k.RetiredAt != "" {
			out = k
			return nil
		}
		var live int64
		if err := tx.Table(m.table(roleActorRoleAssignment)).
			Where("kind_id = ? AND active", kindID).Count(&live).Error; err != nil {
			return err
		}
		if live > 0 {
			return ErrKindHasLiveAssignments
		}
		k.RetiredAt = NowRFC3339()
		k.RetiredBy = actorID
		if err := tx.Table(m.table(roleRoleKind)).Where("id = ?", kindID).
			Updates(map[string]any{"retired_at": k.RetiredAt, "retired_by": k.RetiredBy}).Error; err != nil {
			return err
		}
		out = k
		return nil
	})
	if err != nil {
		return RoleKind{}, err
	}
	return out, nil
}

func (m *userManager) UnretireRoleKind(ctx context.Context, kindID string) (RoleKind, error) {
	var out RoleKind
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		k, err := m.roleKindIn(tx, kindID)
		if err != nil {
			return err
		}
		if k.RetiredAt == "" {
			out = k
			return nil
		}
		k.RetiredAt, k.RetiredBy = "", ""
		if err := tx.Table(m.table(roleRoleKind)).Where("id = ?", kindID).
			Updates(map[string]any{"retired_at": "", "retired_by": ""}).Error; err != nil {
			return err
		}
		out = k
		return nil
	})
	if err != nil {
		return RoleKind{}, err
	}
	return out, nil
}
