package mwanachamaactor

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) RemoveFromGroup(ctx context.Context, actorID, groupID, removedBy string) (models.GroupRemoval, error) {
	if strings.TrimSpace(removedBy) == "" {
		return models.GroupRemoval{}, ErrRemoverRequired
	}
	_, found, err := m.Deregister(ctx, actorID, groupID)
	if err != nil {
		return models.GroupRemoval{}, fmt.Errorf("RemoveFromGroup: %w", err)
	}
	if !found {
		return models.GroupRemoval{}, ErrNotInGroup
	}
	err = m.store.Query(ctx, roleActorRoleAssignment).
		Where("actor_id = ? AND group_id = ? AND active", actorID, groupID).
		Updates(map[string]any{"active": false, "ended_at": NowRFC3339(), "ended_reason": string(EndedByEviction)}).Error
	if err != nil {
		return models.GroupRemoval{}, fmt.Errorf("RemoveFromGroup: %w", err)
	}
	removal := models.GroupRemoval{
		ID:        newSpecID(),
		ActorID:   actorID,
		GroupID:   groupID,
		RemovedBy: removedBy,
		RemovedAt: NowRFC3339(),
	}
	if err := m.store.Insert(ctx, roleGroupRemoval, removal); err != nil {
		return models.GroupRemoval{}, fmt.Errorf("RemoveFromGroup: %w", err)
	}
	return removal, nil
}

func (m *userManager) RemoveFromAllGroups(ctx context.Context, actorID, removedBy string) ([]models.GroupRemoval, error) {
	if strings.TrimSpace(removedBy) == "" {
		return nil, ErrRemoverRequired
	}
	held, err := m.ListGroupsForActor(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("RemoveFromAllGroups: %w", err)
	}
	out := make([]models.GroupRemoval, 0, len(held))
	for _, a := range held {
		removal, err := m.RemoveFromGroup(ctx, actorID, a.GroupID, removedBy)
		if err != nil {
			return out, fmt.Errorf("RemoveFromAllGroups: %w", err)
		}
		out = append(out, removal)
	}
	return out, nil
}

func (m *userManager) ListRemovals(ctx context.Context, actorID string) ([]models.GroupRemoval, error) {
	q := m.store.Query(ctx, roleGroupRemoval).Where("actor_id = ?", actorID)
	out, err := specstore.List[models.GroupRemoval](m.store, q, roleGroupRemoval)
	if err != nil {
		return nil, fmt.Errorf("ListRemovals: %w", err)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RemovedAt < out[j].RemovedAt })
	return out, nil
}
