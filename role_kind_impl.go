// role_kind_impl.go — RoleKind lifecycle for [userManager]. Ported from
// mwanachama-backend-api-gateway's internal/domain/role.go's Repository
// contract and internal/store/{memory,postgres}/role_store_kind.go's
// behavior, minus the custody-log composition DEV-1658 already moved out to
// the gateway's HTTP layer before this port — see that board's row and
// user.go's own doc on RetireRoleKind/UnretireRoleKind for why neither
// writes an audit row here.
package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/gormstore"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

// CreateRoleKind stores a role kind, minting an id when empty. A kind is
// never born retired — RetiredAt/RetiredBy are cleared unconditionally,
// mirroring the gateway's own CreateKind (DEV-1133): a caller-supplied
// ending would create a kind retired the moment it existed.
func (m *userManager) CreateRoleKind(ctx context.Context, k models.RoleKind) (models.RoleKind, error) {
	k.RetiredAt, k.RetiredBy = "", ""
	row, err := gormstore.RoleKindToRow(k)
	if err != nil {
		return models.RoleKind{}, fmt.Errorf("CreateRoleKind: %w", err)
	}
	if err := m.db.WithContext(ctx).Table(m.tables.RoleKinds).Create(&row).Error; err != nil {
		return models.RoleKind{}, fmt.Errorf("CreateRoleKind: %w", err)
	}
	return gormstore.RoleKindFromRow(row)
}

// ListRoleKinds returns every role kind, id order.
func (m *userManager) ListRoleKinds(ctx context.Context) ([]models.RoleKind, error) {
	var rows []gormstore.RoleKindRow
	if err := m.db.WithContext(ctx).Table(m.tables.RoleKinds).Order("id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListRoleKinds: %w", err)
	}
	out := make([]models.RoleKind, 0, len(rows))
	for _, r := range rows {
		k, err := gormstore.RoleKindFromRow(r)
		if err != nil {
			return nil, fmt.Errorf("ListRoleKinds: %w", err)
		}
		out = append(out, k)
	}
	return out, nil
}

// GetRoleKind returns a role kind by id.
func (m *userManager) GetRoleKind(ctx context.Context, id string) (models.RoleKind, error) {
	row, err := m.findRoleKindRow(ctx, id)
	if err != nil {
		return models.RoleKind{}, err
	}
	return gormstore.RoleKindFromRow(row)
}

func (m *userManager) findRoleKindRow(ctx context.Context, id string) (gormstore.RoleKindRow, error) {
	var row gormstore.RoleKindRow
	err := m.db.WithContext(ctx).Table(m.tables.RoleKinds).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.RoleKindRow{}, ErrRoleKindNotFound
	}
	if err != nil {
		return gormstore.RoleKindRow{}, fmt.Errorf("findRoleKindRow: %w", err)
	}
	return row, nil
}

// RetireRoleKind ends a role kind — G229's refusal and the stamp, ported
// unchanged in substance. actorID stamps RetiredBy; the current time stamps
// RetiredAt. Idempotent: re-retiring an already-retired kind returns it
// unchanged rather than overwriting who ended it.
//
// The live-assignment check and the stamp run inside one transaction. Unlike
// the gateway's own Postgres store, this does not take an explicit row lock
// (SELECT ... FOR UPDATE) first — no other method in this package does
// either, and adding the first one here would be a bigger change than this
// port's scope. A grant and a retirement racing on the same kind can still
// interleave into the state G229 exists to prevent; unchanged from every
// other read-then-write in this package (e.g. AssignGroup's upsert), not a
// new gap this port introduces.
func (m *userManager) RetireRoleKind(ctx context.Context, kindID, actorID string) (models.RoleKind, error) {
	var out models.RoleKind
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row gormstore.RoleKindRow
		if err := tx.Table(m.tables.RoleKinds).Where("id = ?", kindID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoleKindNotFound
			}
			return err
		}
		if row.RetiredAt != "" {
			k, err := gormstore.RoleKindFromRow(row)
			out = k
			return err
		}
		var live int64
		if err := tx.Table(m.tables.ActorRoleAssignments).
			Where("kind_id = ? AND active", kindID).Count(&live).Error; err != nil {
			return err
		}
		if live > 0 {
			return ErrKindHasLiveAssignments
		}
		row.RetiredAt = models.NowRFC3339()
		row.RetiredBy = actorID
		if err := tx.Table(m.tables.RoleKinds).Where("id = ?", kindID).
			Updates(map[string]any{"retired_at": row.RetiredAt, "retired_by": row.RetiredBy}).Error; err != nil {
			return err
		}
		k, err := gormstore.RoleKindFromRow(row)
		out = k
		return err
	})
	if err != nil {
		return models.RoleKind{}, err
	}
	return out, nil
}

// UnretireRoleKind reverses a retirement, clearing both fields together.
// Idempotent: un-retiring an already-live kind returns it unchanged.
func (m *userManager) UnretireRoleKind(ctx context.Context, kindID string) (models.RoleKind, error) {
	var out models.RoleKind
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row gormstore.RoleKindRow
		if err := tx.Table(m.tables.RoleKinds).Where("id = ?", kindID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoleKindNotFound
			}
			return err
		}
		if row.RetiredAt == "" {
			k, err := gormstore.RoleKindFromRow(row)
			out = k
			return err
		}
		row.RetiredAt, row.RetiredBy = "", ""
		if err := tx.Table(m.tables.RoleKinds).Where("id = ?", kindID).
			Updates(map[string]any{"retired_at": "", "retired_by": ""}).Error; err != nil {
			return err
		}
		k, err := gormstore.RoleKindFromRow(row)
		out = k
		return err
	})
	if err != nil {
		return models.RoleKind{}, err
	}
	return out, nil
}
