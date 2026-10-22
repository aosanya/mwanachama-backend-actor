package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) SetGroupJoinMode(ctx context.Context, groupID, mode string) (Group, error) {
	if !models.ValidJoinMode(mode) {
		return Group{}, ErrInvalidJoinMode
	}
	return m.setGroupColumn(ctx, groupID, "join_mode", mode, func(g *Group) { g.JoinMode = mode })
}

func (m *userManager) SetGroupHidden(ctx context.Context, groupID string, hidden bool) (Group, error) {
	return m.setGroupColumn(ctx, groupID, "hidden", hidden, func(g *Group) { g.Hidden = hidden })
}

func (m *userManager) setGroupColumn(ctx context.Context, groupID, column string, value any, apply func(*Group)) (Group, error) {
	current, err := m.GetGroup(ctx, groupID)
	if err != nil {
		return Group{}, err
	}
	now := NowRFC3339()
	err = m.store.Query(ctx, roleGroup).Where("id = ? AND deleted = ?", groupID, false).
		Updates(map[string]any{column: value, "last_updated": now}).Error
	if err != nil {
		return Group{}, fmt.Errorf("set group %s: %w", column, err)
	}
	apply(&current)
	current.LastUpdated = now
	return current, nil
}

func (m *userManager) ListJoinableGroups(ctx context.Context, actorID string) ([]Group, error) {
	registered := m.db.WithContext(ctx).Table(m.table(roleActorGroupAssignment)).
		Select("group_id").Where("actor_id = ?", actorID)
	q := m.store.Query(ctx, roleGroup).
		Where("deleted = ? AND hidden = ?", false, false).
		Where("id NOT IN (?)", registered)
	out, err := specstore.List[Group](m.store, q, roleGroup)
	if err != nil {
		return nil, fmt.Errorf("ListJoinableGroups: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *userManager) JoinGroup(ctx context.Context, groupID, actorID, message string) (JoinOutcome, error) {
	return m.join(ctx, groupID, actorID, message, false)
}

func (m *userManager) join(ctx context.Context, groupID, actorID, message string, reachedByCode bool) (JoinOutcome, error) {
	groupID, actorID, message = strings.TrimSpace(groupID), strings.TrimSpace(actorID), strings.TrimSpace(message)
	if groupID == "" || actorID == "" {
		return JoinOutcome{}, fmt.Errorf("%w: a group and an actor are required", ErrInvalidJoinRequest)
	}
	if utf8.RuneCountInString(message) > models.MaxJoinMessageLength {
		return JoinOutcome{}, ErrJoinMessageTooLong
	}
	group, err := m.GetGroup(ctx, groupID)
	if err != nil {
		return JoinOutcome{}, err
	}
	if group.Hidden && !reachedByCode {
		return JoinOutcome{}, ErrGroupNotFound
	}
	if _, err := m.GetActor(ctx, actorID); err != nil {
		return JoinOutcome{}, err
	}
	if err := m.refuseOutsider(m.db.WithContext(ctx), actorID); err != nil {
		return JoinOutcome{}, err
	}
	held, found, err := m.findAssignment(ctx, actorID, groupID)
	if err != nil {
		return JoinOutcome{}, fmt.Errorf("JoinGroup: %w", err)
	}
	if found {
		return JoinOutcome{Status: models.AlreadyAMember, Assignment: &held}, nil
	}
	if group.Open() {
		return m.joinOpen(ctx, groupID, actorID)
	}
	return m.askToJoin(ctx, groupID, actorID, message, true)
}

func (m *userManager) joinOpen(ctx context.Context, groupID, actorID string) (JoinOutcome, error) {
	now := NowRFC3339()
	var assignment ActorGroupAssignment
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		assignment, err = m.register(ctx, tx, actorID, groupID, now)
		if err != nil {
			return err
		}
		return m.settlePendingJoinRequest(tx, groupID, actorID, now)
	})
	if err != nil {
		return JoinOutcome{}, fmt.Errorf("JoinGroup: %w", err)
	}
	return JoinOutcome{Status: models.JoinedNow, Assignment: &assignment}, nil
}

func (m *userManager) settlePendingJoinRequest(tx *gorm.DB, groupID, actorID, now string) error {
	return tx.Table(m.table(roleGroupJoinRequest)).
		Where("group_id = ? AND actor_id = ? AND state = ?", groupID, actorID, models.JoinRequestPending).
		Updates(map[string]any{
			"state":          models.JoinRequestApproved,
			"silent":         false,
			"decided_by":     "",
			"decided_at":     now,
			"decline_reason": "",
			"last_updated":   now,
		}).Error
}

func (m *userManager) askToJoin(ctx context.Context, groupID, actorID, message string, retryOnRace bool) (JoinOutcome, error) {
	var out GroupJoinRequest
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, found, err := m.findJoinRequest(ctx, tx, groupID, actorID)
		if err != nil {
			return err
		}
		now := NowRFC3339()
		if !found {
			out = GroupJoinRequest{
				ID:          newSpecID(),
				GroupID:     groupID,
				ActorID:     actorID,
				State:       models.JoinRequestPending,
				Message:     message,
				RequestedAt: now,
				AskedAt:     now,
				LastUpdated: now,
			}
			row, err := encode(m.object(roleGroupJoinRequest), out)
			if err != nil {
				return err
			}
			return tx.Table(m.table(roleGroupJoinRequest)).Create(row).Error
		}
		out = askedAgain(existing, message, now)
		return tx.Table(m.table(roleGroupJoinRequest)).Where("id = ?", out.ID).
			Updates(map[string]any{
				"state":          out.State,
				"appeal":         out.Appeal,
				"message":        out.Message,
				"silent":         out.Silent,
				"requested_at":   out.RequestedAt,
				"asked_at":       out.AskedAt,
				"decided_by":     out.DecidedBy,
				"decided_at":     out.DecidedAt,
				"decline_reason": out.DeclineReason,
				"last_updated":   out.LastUpdated,
			}).Error
	})
	if err != nil {
		if retryOnRace && errors.Is(classifyDuplicateID(err), ErrDuplicateID) {
			return m.askToJoin(ctx, groupID, actorID, message, false)
		}
		return JoinOutcome{}, fmt.Errorf("JoinGroup: %w", err)
	}
	return JoinOutcome{Status: models.JoinRequested, Request: &out}, nil
}

func askedAgain(r GroupJoinRequest, message, now string) GroupJoinRequest {
	switch r.State {
	case models.JoinRequestDeclined:
		r.Appeal = true
	case models.JoinRequestApproved:
		r.Appeal = false
		r.RequestedAt = now
	}
	r.State = models.JoinRequestPending
	r.Message = message
	r.Silent = false
	r.DecidedBy = ""
	r.DecidedAt = ""
	r.DeclineReason = ""
	r.AskedAt = now
	r.LastUpdated = now
	return r
}

func (m *userManager) findJoinRequest(ctx context.Context, tx *gorm.DB, groupID, actorID string) (GroupJoinRequest, bool, error) {
	var out GroupJoinRequest
	q := tx.WithContext(ctx).Table(m.table(roleGroupJoinRequest)).
		Where("group_id = ? AND actor_id = ?", groupID, actorID)
	err := m.store.Take(q, roleGroupJoinRequest, &out, ErrJoinRequestNotFound)
	if errors.Is(err, ErrJoinRequestNotFound) {
		return GroupJoinRequest{}, false, nil
	}
	if err != nil {
		return GroupJoinRequest{}, false, err
	}
	return out, true, nil
}

func (m *userManager) register(ctx context.Context, tx *gorm.DB, actorID, groupID, now string) (ActorGroupAssignment, error) {
	a := ActorGroupAssignment{ActorID: actorID, GroupID: groupID, CreatedAt: now, LastUpdated: now}
	row, err := encode(m.object(roleActorGroupAssignment), a)
	if err != nil {
		return ActorGroupAssignment{}, err
	}
	err = tx.WithContext(ctx).Table(m.table(roleActorGroupAssignment)).
		Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error
	if err != nil {
		return ActorGroupAssignment{}, err
	}
	var held ActorGroupAssignment
	q := tx.WithContext(ctx).Table(m.table(roleActorGroupAssignment)).
		Where("actor_id = ? AND group_id = ?", actorID, groupID)
	if err := m.store.Take(q, roleActorGroupAssignment, &held, errAssignmentNotHeld); err != nil {
		return ActorGroupAssignment{}, err
	}
	return held, nil
}

func (m *userManager) GetJoinRequest(ctx context.Context, id string) (GroupJoinRequest, error) {
	var out GroupJoinRequest
	q := m.store.Query(ctx, roleGroupJoinRequest).Where("id = ?", id)
	if err := m.store.Take(q, roleGroupJoinRequest, &out, ErrJoinRequestNotFound); err != nil {
		if errors.Is(err, ErrJoinRequestNotFound) {
			return GroupJoinRequest{}, ErrJoinRequestNotFound
		}
		return GroupJoinRequest{}, fmt.Errorf("GetJoinRequest: %w", err)
	}
	return out, nil
}

func (m *userManager) ListJoinRequestsForGroup(ctx context.Context, groupID string, pendingOnly bool) ([]GroupJoinRequest, error) {
	q := m.store.Query(ctx, roleGroupJoinRequest).Where("group_id = ?", groupID)
	if pendingOnly {
		q = q.Where("state = ?", models.JoinRequestPending)
	}
	return m.listJoinRequests(q, "ListJoinRequestsForGroup")
}

func (m *userManager) ListJoinRequestsForActor(ctx context.Context, actorID string) ([]GroupJoinRequest, error) {
	q := m.store.Query(ctx, roleGroupJoinRequest).Where("actor_id = ?", actorID)
	return m.listJoinRequests(q, "ListJoinRequestsForActor")
}

func (m *userManager) listJoinRequests(q *gorm.DB, op string) ([]GroupJoinRequest, error) {
	out, err := specstore.List[GroupJoinRequest](m.store, q, roleGroupJoinRequest)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RequestedAt != out[j].RequestedAt {
			return out[i].RequestedAt < out[j].RequestedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (m *userManager) ApproveJoinRequest(ctx context.Context, id, decidedBy string) (GroupJoinRequest, error) {
	return m.decide(ctx, id, decidedBy, models.JoinRequestApproved, false, "", m.mustLead)
}

func (m *userManager) DeclineJoinRequest(ctx context.Context, id, decidedBy string, silent bool, reason string) (GroupJoinRequest, error) {
	return m.decline(ctx, id, decidedBy, silent, reason, m.mustLead)
}

func (m *userManager) OperatorApproveJoinRequest(ctx context.Context, id, decidedBy string) (GroupJoinRequest, error) {
	return m.decide(ctx, id, decidedBy, models.JoinRequestApproved, false, "", vouchedByHost)
}

func (m *userManager) OperatorDeclineJoinRequest(ctx context.Context, id, decidedBy string, silent bool, reason string) (GroupJoinRequest, error) {
	return m.decline(ctx, id, decidedBy, silent, reason, vouchedByHost)
}

type deciderCheck func(tx *gorm.DB, groupID, actorID string) error

func vouchedByHost(*gorm.DB, string, string) error {
	return nil
}

func (m *userManager) decline(ctx context.Context, id, decidedBy string, silent bool, reason string, mayDecide deciderCheck) (GroupJoinRequest, error) {
	reason = strings.TrimSpace(reason)
	if reason != "" && silent {
		return GroupJoinRequest{}, ErrQuietDeclineReason
	}
	if utf8.RuneCountInString(reason) > models.MaxDeclineReasonLength {
		return GroupJoinRequest{}, ErrDeclineReasonTooLong
	}
	return m.decide(ctx, id, decidedBy, models.JoinRequestDeclined, silent, reason, mayDecide)
}

func (m *userManager) decide(ctx context.Context, id, decidedBy, state string, silent bool, reason string, mayDecide deciderCheck) (GroupJoinRequest, error) {
	decidedBy = strings.TrimSpace(decidedBy)
	if decidedBy == "" {
		return GroupJoinRequest{}, fmt.Errorf("%w: a decision needs whoever made it", ErrInvalidJoinRequest)
	}
	req, err := m.GetJoinRequest(ctx, id)
	if err != nil {
		return GroupJoinRequest{}, err
	}
	now := NowRFC3339()
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := mayDecide(tx, req.GroupID, decidedBy); err != nil {
			return err
		}
		taken := tx.Table(m.table(roleGroupJoinRequest)).
			Where("id = ? AND state = ?", id, models.JoinRequestPending).
			Updates(map[string]any{
				"state":          state,
				"silent":         silent,
				"decided_by":     decidedBy,
				"decided_at":     now,
				"decline_reason": reason,
				"last_updated":   now,
			})
		if taken.Error != nil {
			return taken.Error
		}
		if taken.RowsAffected == 0 {
			return ErrJoinRequestDecided
		}
		if err := m.recordDecision(tx, req, state, silent, decidedBy, now, reason); err != nil {
			return err
		}
		if state != models.JoinRequestApproved {
			return nil
		}
		_, err := m.register(ctx, tx, req.ActorID, req.GroupID, now)
		return err
	})
	if errors.Is(err, ErrJoinRequestDecided) || errors.Is(err, ErrNotGroupLeader) {
		return GroupJoinRequest{}, err
	}
	if err != nil {
		return GroupJoinRequest{}, fmt.Errorf("decide join request: %w", err)
	}
	req.State = state
	req.Silent = silent
	req.DecidedBy = decidedBy
	req.DecidedAt = now
	req.DeclineReason = reason
	req.LastUpdated = now
	return req, nil
}
