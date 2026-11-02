package mwanachamaactor

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) GetActorStanding(ctx context.Context, actorID string) (models.ActorStanding, error) {
	actor, err := m.GetActor(ctx, actorID)
	if err != nil {
		return models.ActorStanding{}, err
	}
	registrations, err := m.ListGroupsForActor(ctx, actor.ID)
	if err != nil {
		return models.ActorStanding{}, fmt.Errorf("GetActorStanding: %w", err)
	}
	seats, err := m.ListRoleAssignmentsForActor(ctx, actor.ID, false)
	if err != nil {
		return models.ActorStanding{}, fmt.Errorf("GetActorStanding: %w", err)
	}
	kinds, err := m.ListRoleKinds(ctx)
	if err != nil {
		return models.ActorStanding{}, fmt.Errorf("GetActorStanding: %w", err)
	}
	kindsByID := make(map[string]RoleKind, len(kinds))
	for _, k := range kinds {
		kindsByID[k.ID] = k
	}
	groupIDs := make([]string, 0, len(registrations)+len(seats))
	granterIDs := make([]string, 0, len(seats))
	for _, r := range registrations {
		groupIDs = append(groupIDs, r.GroupID)
	}
	for _, s := range seats {
		groupIDs = append(groupIDs, s.GroupID)
		if s.GrantedBy != "" {
			granterIDs = append(granterIDs, s.GrantedBy)
		}
	}
	groups, err := m.groupsByID(ctx, groupIDs)
	if err != nil {
		return models.ActorStanding{}, fmt.Errorf("GetActorStanding: %w", err)
	}
	granters, err := m.GetActors(ctx, granterIDs)
	if err != nil {
		return models.ActorStanding{}, fmt.Errorf("GetActorStanding: %w", err)
	}
	granterNames := make(map[string]string, len(granters))
	for _, g := range granters {
		granterNames[g.ID] = g.DisplayName
	}

	out := models.ActorStanding{
		ActorID:       actor.ID,
		DisplayName:   actor.DisplayName,
		Registrations: []models.StandingRegistration{},
		Seats:         []models.StandingSeat{},
	}
	for _, r := range registrations {
		if g, ok := groups[r.GroupID]; ok && !g.Deleted {
			out.Registrations = append(out.Registrations, models.StandingRegistration{
				GroupID: r.GroupID, GroupName: g.Name, Since: r.CreatedAt,
			})
		}
	}
	for _, s := range seats {
		kind := kindsByID[s.KindID]
		out.Seats = append(out.Seats, models.StandingSeat{
			ID:               s.ID,
			GroupID:          s.GroupID,
			GroupName:        groups[s.GroupID].Name,
			KindID:           s.KindID,
			RoleName:         kind.Name,
			DefinitionRoleID: kind.DefinitionRoleID,
			LeadsGroup:       kind.LeadsGroup,
			Staff:            kind.Staff,
			GrantedAt:        s.GrantedAt,
			GrantedBy:        s.GrantedBy,
			GrantedByName:    granterNames[s.GrantedBy],
			EndedAt:          s.EndedAt,
			EndedReason:      s.EndedReason,
		})
	}
	return out, nil
}

func (m *userManager) groupsByID(ctx context.Context, ids []string) (map[string]Group, error) {
	out := map[string]Group{}
	if len(ids) == 0 {
		return out, nil
	}
	q := m.store.Query(ctx, roleGroup).Where("id IN ?", ids)
	groups, err := specstore.List[Group](m.store, q, roleGroup)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		out[g.ID] = g
	}
	return out, nil
}
