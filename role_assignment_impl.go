// role_assignment_impl.go — ActorRoleAssignment lifecycle for [userManager].
// Ported from mwanachama-backend-api-gateway's internal/domain/role.go's
// Repository contract and internal/store/{memory,postgres}/role_store.go's
// behavior, minus the custody-log composition DEV-1658 moved to the
// gateway's HTTP layer before this port.
package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/gormstore"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

// GrantRole assigns a role kind to an actor at a group. role_kind_id has no
// foreign key in this package's own tables — same shape as
// ActorGroupAssignmentRow's actor/group ids — so the kind's existence and
// retirement are checked explicitly, in the same transaction as the insert,
// for the reason RetireRoleKind's does: a retirement and a grant racing on
// the same kind must not both land in the state G229 refuses.
func (m *userManager) GrantRole(ctx context.Context, a ActorRoleAssignment) (ActorRoleAssignment, error) {
	var out ActorRoleAssignment
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var kindRow gormstore.RoleKindRow
		if err := tx.Table(m.tables.RoleKinds).Where("id = ?", a.KindID).First(&kindRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRoleKindNotFound
			}
			return err
		}
		if kindRow.RetiredAt != "" {
			return ErrKindRetired
		}
		a.GrantedAt = NowRFC3339()
		a.Active = true
		a.EndedAt, a.EndedReason = "", ""
		row := gormstore.ActorRoleAssignmentToRow(a)
		if err := tx.Table(m.tables.ActorRoleAssignments).Create(&row).Error; err != nil {
			return err
		}
		out = gormstore.ActorRoleAssignmentFromRow(row)
		return nil
	})
	if err != nil {
		return ActorRoleAssignment{}, fmt.Errorf("GrantRole: %w", err)
	}
	return out, nil
}

// GetRoleAssignment returns one assignment by id, active or not.
func (m *userManager) GetRoleAssignment(ctx context.Context, id string) (ActorRoleAssignment, error) {
	row, err := m.findRoleAssignmentRow(ctx, id)
	if err != nil {
		return ActorRoleAssignment{}, err
	}
	return gormstore.ActorRoleAssignmentFromRow(row), nil
}

func (m *userManager) findRoleAssignmentRow(ctx context.Context, id string) (gormstore.ActorRoleAssignmentRow, error) {
	var row gormstore.ActorRoleAssignmentRow
	err := m.db.WithContext(ctx).Table(m.tables.ActorRoleAssignments).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.ActorRoleAssignmentRow{}, ErrAssignmentNotFound
	}
	if err != nil {
		return gormstore.ActorRoleAssignmentRow{}, fmt.Errorf("findRoleAssignmentRow: %w", err)
	}
	return row, nil
}

// RevokeRole deactivates an assignment (an operator removing the holder).
func (m *userManager) RevokeRole(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByRevocation)
}

// StepDownRole deactivates an assignment at the holder's own request.
func (m *userManager) StepDownRole(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByResignation)
}

// EndRoleOnEviction deactivates an assignment because an operator ended the
// holder's membership at the seat's group.
func (m *userManager) EndRoleOnEviction(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByEviction)
}

// EndRoleOnDeparture deactivates an assignment because the holder ended
// their own membership at the seat's group.
func (m *userManager) EndRoleOnDeparture(ctx context.Context, assignmentID string) error {
	return m.deactivateRole(ctx, assignmentID, EndedByDeparture)
}

// deactivateRole is the one path every ender takes, so a future fifth one
// cannot drift from how the first four flip the three fields together.
//
// Re-ending an already-inactive assignment keeps the FIRST reason and
// timestamp — the seat ended once, and a later call is a no-op on a row
// that is already closed. A miss (0 rows affected on an already-inactive
// row) is disambiguated with an explicit existence lookup rather than
// reported as a 404 the caller would not understand — mirrors the gateway's
// own postgres store.
func (m *userManager) deactivateRole(ctx context.Context, id string, reason models.EndReason) error {
	res := m.db.WithContext(ctx).Table(m.tables.ActorRoleAssignments).
		Where("id = ? AND active", id).
		Updates(map[string]any{"active": false, "ended_at": NowRFC3339(), "ended_reason": string(reason)})
	if res.Error != nil {
		return fmt.Errorf("deactivateRole: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		if _, err := m.findRoleAssignmentRow(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// ListRoleAssignmentsForGroup returns assignments at a group, (granted_at,
// id) order.
func (m *userManager) ListRoleAssignmentsForGroup(ctx context.Context, groupID string, activeOnly bool) ([]ActorRoleAssignment, error) {
	return m.listRoleAssignments(ctx, "group_id = ?", groupID, activeOnly)
}

// ListRoleAssignmentsForActor returns every assignment an actor holds,
// across every group, (granted_at, id) order.
func (m *userManager) ListRoleAssignmentsForActor(ctx context.Context, actorID string, activeOnly bool) ([]ActorRoleAssignment, error) {
	return m.listRoleAssignments(ctx, "actor_id = ?", actorID, activeOnly)
}

func (m *userManager) listRoleAssignments(ctx context.Context, whereCol string, whereArg string, activeOnly bool) ([]ActorRoleAssignment, error) {
	q := m.db.WithContext(ctx).Table(m.tables.ActorRoleAssignments).Where(whereCol, whereArg)
	if activeOnly {
		q = q.Where("active")
	}
	var rows []gormstore.ActorRoleAssignmentRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("listRoleAssignments: %w", err)
	}
	out := make([]ActorRoleAssignment, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.ActorRoleAssignmentFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GrantedAt != out[j].GrantedAt {
			return out[i].GrantedAt < out[j].GrantedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
