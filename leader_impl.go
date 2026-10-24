package mwanachamaactor

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func (m *userManager) SetRoleKindLeadsGroup(ctx context.Context, kindID string, leads bool) (RoleKind, error) {
	var out RoleKind
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		k, err := m.roleKindIn(tx, kindID)
		if err != nil {
			return err
		}
		if err := tx.Table(m.table(roleRoleKind)).Where("id = ?", kindID).
			Updates(map[string]any{"leads_group": leads}).Error; err != nil {
			return err
		}
		k.LeadsGroup = leads
		out = k
		return nil
	})
	if err != nil {
		return RoleKind{}, err
	}
	return out, nil
}

func (m *userManager) leadingSeats(tx *gorm.DB, actorID string) *gorm.DB {
	leadingKinds := tx.Table(m.table(roleRoleKind)).Select("id").
		Where("leads_group = ?", true).
		Where("retired_at = ''")
	return tx.Table(m.table(roleActorRoleAssignment)).Select("group_id").
		Where("actor_id = ? AND active", actorID).
		Where("kind_id IN (?)", leadingKinds)
}

func (m *userManager) IsGroupLeader(ctx context.Context, groupID, actorID string) (bool, error) {
	leads, err := m.leadsIn(m.db.WithContext(ctx), groupID, actorID)
	if err != nil {
		return false, fmt.Errorf("IsGroupLeader: %w", err)
	}
	return leads, nil
}

func (m *userManager) leadsIn(tx *gorm.DB, groupID, actorID string) (bool, error) {
	groupID, actorID = strings.TrimSpace(groupID), strings.TrimSpace(actorID)
	if groupID == "" || actorID == "" {
		return false, nil
	}
	var seats int64
	err := m.leadingSeats(tx, actorID).Where("group_id = ?", groupID).Count(&seats).Error
	if err != nil {
		return false, err
	}
	return seats > 0, nil
}

func (m *userManager) mustLead(tx *gorm.DB, groupID, actorID string) error {
	leads, err := m.leadsIn(tx, groupID, actorID)
	if err != nil {
		return err
	}
	if !leads {
		return ErrNotGroupLeader
	}
	return nil
}

func (m *userManager) ListLedGroups(ctx context.Context, actorID string) ([]Group, error) {
	q := m.store.Query(ctx, roleGroup).
		Where("deleted = ?", false).
		Where("id IN (?)", m.leadingSeats(m.db.WithContext(ctx), strings.TrimSpace(actorID)))
	out, err := specstore.List[Group](m.store, q, roleGroup)
	if err != nil {
		return nil, fmt.Errorf("ListLedGroups: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
