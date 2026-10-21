package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func normalizeGroupCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (m *userManager) groupByCode(ctx context.Context, code string) (Group, error) {
	code = normalizeGroupCode(code)
	if code == "" {
		return Group{}, ErrGroupNotFound
	}
	var out Group
	q := m.store.Query(ctx, roleGroup).Where("code = ? AND deleted = ?", code, false)
	if err := m.store.Take(q, roleGroup, &out, ErrGroupNotFound); err != nil {
		if errors.Is(err, ErrGroupNotFound) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, fmt.Errorf("find group by code: %w", err)
	}
	return out, nil
}

func (m *userManager) FindGroupByCode(ctx context.Context, code, actorID string) (GroupByCode, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return GroupByCode{}, fmt.Errorf("%w: an actor is required", ErrInvalidJoinRequest)
	}
	group, err := m.groupByCode(ctx, code)
	if err != nil {
		return GroupByCode{}, err
	}
	out := GroupByCode{Group: group, JoinMode: group.JoinMode, State: models.CallerIsOutside}
	_, member, err := m.findAssignment(ctx, actorID, group.ID)
	if err != nil {
		return GroupByCode{}, fmt.Errorf("FindGroupByCode: %w", err)
	}
	if member {
		out.State = models.CallerIsMember
		return out, nil
	}
	req, asked, err := m.findJoinRequest(ctx, m.db, group.ID, actorID)
	if err != nil {
		return GroupByCode{}, fmt.Errorf("FindGroupByCode: %w", err)
	}
	if !asked {
		return out, nil
	}
	switch req.State {
	case models.JoinRequestPending:
		out.State = models.CallerRequested
		out.Request = &req
	case models.JoinRequestDeclined:
		out.State = models.CallerDeclined
		out.Request = &req
	}
	return out, nil
}

func (m *userManager) JoinGroupByCode(ctx context.Context, code, actorID, message string) (JoinOutcome, error) {
	group, err := m.groupByCode(ctx, code)
	if err != nil {
		return JoinOutcome{}, err
	}
	return m.join(ctx, group.ID, actorID, message, true)
}
