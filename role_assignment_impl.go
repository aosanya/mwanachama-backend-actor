package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) GrantRole(ctx context.Context, a ActorRoleAssignment) (ActorRoleAssignment, error) {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		kind, err := m.roleKindIn(tx, a.KindID)
		if err != nil {
			return err
		}
		if kind.RetiredAt != "" {
			return ErrKindRetired
		}
		a.ID = newSpecID()
		a.GrantedAt = NowRFC3339()
		a.Active = true
		a.EndedAt, a.EndedReason = "", ""
		row, err := m.store.Row(roleActorRoleAssignment, a)
		if err != nil {
			return err
		}
		return m.from(tx.WithContext(ctx), roleActorRoleAssignment).Create(row).Error
	})
	if err != nil {
		return ActorRoleAssignment{}, fmt.Errorf("GrantRole: %w", err)
	}
	return a, nil
}

func (m *userManager) GetRoleAssignment(ctx context.Context, id string) (ActorRoleAssignment, error) {
	var out ActorRoleAssignment
	q := m.store.Query(ctx, roleActorRoleAssignment).Where("id = ?", id)
	if err := m.store.Take(q, roleActorRoleAssignment, &out, ErrAssignmentNotFound); err != nil {
		if errors.Is(err, ErrAssignmentNotFound) {
			return ActorRoleAssignment{}, ErrAssignmentNotFound
		}
		return ActorRoleAssignment{}, fmt.Errorf("GetRoleAssignment: %w", err)
	}
	return out, nil
}

func (m *userManager) RevokeRole(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByRevocation)
}

func (m *userManager) StepDownRole(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByResignation)
}

func (m *userManager) EndRoleOnEviction(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByEviction)
}

func (m *userManager) EndRoleOnDeparture(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByDeparture)
}

func (m *userManager) deactivateRole(ctx context.Context, id string, reason models.EndReason) error {
	res := m.store.Query(ctx, roleActorRoleAssignment).
		Where("id = ? AND active", id).
		Updates(map[string]any{"active": false, "ended_at": NowRFC3339(), "ended_reason": string(reason)})
	if res.Error != nil {
		return fmt.Errorf("deactivateRole: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		if _, err := m.GetRoleAssignment(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (m *userManager) ListRoleAssignmentsForGroup(ctx context.Context, groupID string, activeOnly bool) ([]ActorRoleAssignment, error) {
	return m.listRoleAssignments(ctx, "group_id = ?", groupID, activeOnly)
}

func (m *userManager) ListRoleAssignmentsForActor(ctx context.Context, actorID string, activeOnly bool) ([]ActorRoleAssignment, error) {
	return m.listRoleAssignments(ctx, "actor_id = ?", actorID, activeOnly)
}

func (m *userManager) listRoleAssignments(ctx context.Context, whereCol string, whereArg string, activeOnly bool) ([]ActorRoleAssignment, error) {
	q := m.store.Query(ctx, roleActorRoleAssignment).Where(whereCol, whereArg)
	if activeOnly {
		q = q.Where("active")
	}
	out, err := specstore.List[ActorRoleAssignment](m.store, q, roleActorRoleAssignment)
	if err != nil {
		return nil, fmt.Errorf("listRoleAssignments: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GrantedAt != out[j].GrantedAt {
			return out[i].GrantedAt < out[j].GrantedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
