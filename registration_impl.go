package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/datatypes"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

var errAssignmentNotHeld = errors.New("actor: no assignment")

func (m *userManager) findAssignment(ctx context.Context, actorID, groupID string) (ActorGroupAssignment, bool, error) {
	var out ActorGroupAssignment
	q := m.store.Query(ctx, roleActorGroupAssignment).
		Where("actor_id = ? AND group_id = ?", actorID, groupID)
	err := m.store.Take(q, roleActorGroupAssignment, &out, errAssignmentNotHeld)
	if errors.Is(err, errAssignmentNotHeld) {
		return ActorGroupAssignment{}, false, nil
	}
	if err != nil {
		return ActorGroupAssignment{}, false, err
	}
	return out, true, nil
}

func (m *userManager) AssignGroup(ctx context.Context, r ActorGroupAssignment) (ActorGroupAssignment, error) {
	properties := models.DefaultActorGroupAssignmentProperties()
	if err := models.ValidateAttributes(properties, r.Attributes); err != nil {
		return ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
	}

	existing, found, err := m.findAssignment(ctx, r.ActorID, r.GroupID)
	if err != nil {
		return ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
	}
	if found {
		r.CreatedAt = existing.CreatedAt
		r.Priority = models.PriorityOrDefault(existing.Priority)
	} else {
		r.CreatedAt = NowRFC3339()
		r.Priority = models.PriorityNormal
	}
	r.LastUpdated = NowRFC3339()
	r.Deleted = false

	if found {
		row, err := encode(m.object(roleActorGroupAssignment), r)
		if err != nil {
			return ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
		}
		err = m.store.Query(ctx, roleActorGroupAssignment).
			Where("actor_id = ? AND group_id = ?", r.ActorID, r.GroupID).
			Updates(map[string]any{"last_updated": row["last_updated"], "attributes": row["attributes"]}).Error
		if err != nil {
			return ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
		}
		return r, nil
	}

	var actorExists int64
	if cerr := m.store.Query(ctx, roleActor).Where("id = ?", r.ActorID).Count(&actorExists).Error; cerr != nil {
		return ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", cerr)
	}
	if actorExists == 0 {
		return ActorGroupAssignment{}, ErrActorNotFound
	}
	if err := m.checkUniqueAttributes(ctx, m.table(roleActorGroupAssignment), properties, r.Attributes); err != nil {
		return ActorGroupAssignment{}, err
	}
	if err := m.store.Insert(ctx, roleActorGroupAssignment, r); err != nil {
		return ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
	}
	return r, nil
}

func (m *userManager) Deregister(ctx context.Context, actorID, groupID string) (ActorGroupAssignment, bool, error) {
	held, found, err := m.findAssignment(ctx, actorID, groupID)
	if err != nil {
		return ActorGroupAssignment{}, false, fmt.Errorf("Deregister: %w", err)
	}
	if !found {
		return ActorGroupAssignment{}, false, nil
	}
	err = m.store.Query(ctx, roleActorGroupAssignment).
		Where("actor_id = ? AND group_id = ?", actorID, groupID).
		Delete(map[string]any{}).Error
	if err != nil {
		return ActorGroupAssignment{}, false, fmt.Errorf("Deregister: %w", err)
	}
	return held, true, nil
}

func (m *userManager) ListGroupsForActor(ctx context.Context, actorID string) ([]ActorGroupAssignment, error) {
	q := m.store.Query(ctx, roleActorGroupAssignment).Where("actor_id = ?", actorID)
	out, err := specstore.List[ActorGroupAssignment](m.store, q, roleActorGroupAssignment)
	if err != nil {
		return nil, fmt.Errorf("ListGroupsForActor: %w", err)
	}
	defaultPriorities(out)
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].GroupID < out[j].GroupID
	})
	return out, nil
}

func (m *userManager) ListActorsForGroup(ctx context.Context, groupID string) ([]ActorGroupAssignment, error) {
	q := m.store.Query(ctx, roleActorGroupAssignment).Where("group_id = ?", groupID).
		Where("actor_id NOT IN (?)", m.outsiderActors(m.db.WithContext(ctx)))
	out, err := specstore.List[ActorGroupAssignment](m.store, q, roleActorGroupAssignment)
	if err != nil {
		return nil, fmt.Errorf("ListActorsForGroup: %w", err)
	}
	defaultPriorities(out)
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ActorID < out[j].ActorID
	})
	return out, nil
}

func (m *userManager) HomeCounts(ctx context.Context) (map[string]int, error) {
	q := m.store.Query(ctx, roleActorGroupAssignment).
		Where(datatypes.JSONQuery("attributes").Equals(true, "is_home")).
		Where("actor_id NOT IN (?)", m.outsiderActors(m.db.WithContext(ctx)))
	rows, err := specstore.List[ActorGroupAssignment](m.store, q, roleActorGroupAssignment)
	if err != nil {
		return nil, fmt.Errorf("HomeCounts: %w", err)
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.GroupID]++
	}
	return out, nil
}

func (m *userManager) ListMyGroups(ctx context.Context, actorID string) ([]MyGroup, error) {
	out := []MyGroup{}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return out, nil
	}
	held, err := m.ListGroupsForActor(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("ListMyGroups: %w", err)
	}
	if len(held) == 0 {
		return out, nil
	}
	ids := make([]string, len(held))
	for i, a := range held {
		ids[i] = a.GroupID
	}
	q := m.store.Query(ctx, roleGroup).Where("deleted = ?", false).Where("id IN ?", ids)
	groups, err := specstore.List[Group](m.store, q, roleGroup)
	if err != nil {
		return nil, fmt.Errorf("ListMyGroups: %w", err)
	}
	byID := make(map[string]Group, len(groups))
	for _, g := range groups {
		byID[g.ID] = g
	}
	for _, a := range held {
		if g, ok := byID[a.GroupID]; ok {
			out = append(out, MyGroup{Group: g, Priority: models.PriorityOrDefault(a.Priority)})
		}
	}
	return out, nil
}

func defaultPriorities(rows []ActorGroupAssignment) {
	for i := range rows {
		rows[i].Priority = models.PriorityOrDefault(rows[i].Priority)
	}
}

func (m *userManager) SetMyGroupPriority(ctx context.Context, groupID, actorID, priority string) (ActorGroupAssignment, error) {
	priority = strings.TrimSpace(priority)
	if !models.ValidPriority(priority) {
		return ActorGroupAssignment{}, ErrInvalidPriority
	}
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return ActorGroupAssignment{}, ErrGroupNotFound
	}
	held, found, err := m.findAssignment(ctx, actorID, groupID)
	if err != nil {
		return ActorGroupAssignment{}, fmt.Errorf("SetMyGroupPriority: %w", err)
	}
	if !found {
		return ActorGroupAssignment{}, ErrGroupNotFound
	}
	held.Priority = priority
	held.LastUpdated = NowRFC3339()
	row, err := encode(m.object(roleActorGroupAssignment), held)
	if err != nil {
		return ActorGroupAssignment{}, fmt.Errorf("SetMyGroupPriority: %w", err)
	}
	err = m.store.Query(ctx, roleActorGroupAssignment).
		Where("actor_id = ? AND group_id = ?", actorID, groupID).
		Updates(map[string]any{"priority": row["priority"], "last_updated": row["last_updated"]}).Error
	if err != nil {
		return ActorGroupAssignment{}, fmt.Errorf("SetMyGroupPriority: %w", err)
	}
	return held, nil
}
