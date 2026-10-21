package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func (m *userManager) recordDecision(tx *gorm.DB, req GroupJoinRequest, state string, silent bool, decidedBy, now, reason string) error {
	d := JoinDecision{
		ID:        newSpecID(),
		RequestID: req.ID,
		GroupID:   req.GroupID,
		ActorID:   req.ActorID,
		State:     state,
		Silent:    silent,
		DecidedBy: decidedBy,
		DecidedAt: now,
		Reason:    reason,
	}
	row, err := encode(m.object(roleJoinDecision), d)
	if err != nil {
		return err
	}
	return tx.Table(m.table(roleJoinDecision)).Create(row).Error
}

func (m *userManager) GetMyJoinDecision(ctx context.Context, decisionID, actorID string) (JoinDecision, error) {
	decisionID, actorID = strings.TrimSpace(decisionID), strings.TrimSpace(actorID)
	if decisionID == "" || actorID == "" {
		return JoinDecision{}, ErrJoinDecisionNotFound
	}
	var out JoinDecision
	q := m.store.Query(ctx, roleJoinDecision).
		Where("id = ? AND actor_id = ? AND silent = ?", decisionID, actorID, false)
	if err := m.store.Take(q, roleJoinDecision, &out, ErrJoinDecisionNotFound); err != nil {
		if errors.Is(err, ErrJoinDecisionNotFound) {
			return JoinDecision{}, ErrJoinDecisionNotFound
		}
		return JoinDecision{}, fmt.Errorf("GetMyJoinDecision: %w", err)
	}
	return out, nil
}

func (m *userManager) ListJoinDecisions(ctx context.Context, requestID string) ([]JoinDecision, error) {
	q := m.store.Query(ctx, roleJoinDecision).Where("request_id = ?", requestID)
	out, err := specstore.List[JoinDecision](m.store, q, roleJoinDecision)
	if err != nil {
		return nil, fmt.Errorf("ListJoinDecisions: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DecidedAt != out[j].DecidedAt {
			return out[i].DecidedAt < out[j].DecidedAt
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
