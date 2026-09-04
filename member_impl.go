// member_impl.go — Member CRUD implementation for [userManager]. Ported
// from mwanachama-backend-api-gateway's internal/store/postgres/member_store.go
// and internal/store/memory/member_store.go (the identity half; the
// registration half is in registration_impl.go, mirroring the gateway's own
// member_store.go / member_registration_store.go split).
package mwanachamauser

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/aosanya/mwanachama-backend-shared/entitygraph"
)

// CreateMember creates a Member entity.
func (m *userManager) CreateMember(ctx context.Context, mem Member) (Member, error) {
	if mem.CreatedAt == "" {
		mem.CreatedAt = nowRFC3339()
	}
	created, err := m.dm.CreateEntity(ctx, entitygraph.CreateEntityRequest{
		TypeID:     memberTypeID,
		Properties: memberToProperties(mem),
	})
	if err != nil {
		return Member{}, fmt.Errorf("CreateMember: %w", err)
	}
	return memberFromEntity(created), nil
}

// GetMember reads a single Member entity.
func (m *userManager) GetMember(ctx context.Context, id string) (Member, error) {
	e, err := m.dm.GetEntity(ctx, id)
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Member{}, ErrMemberNotFound
		}
		return Member{}, fmt.Errorf("GetMember: %w", err)
	}
	if e.TypeID != memberTypeID {
		return Member{}, ErrMemberNotFound
	}
	return memberFromEntity(e), nil
}

// GetMembers returns the Members for the given ids, sorted by id. Ids with
// no member are skipped rather than erroring.
func (m *userManager) GetMembers(ctx context.Context, ids []string) ([]Member, error) {
	out := []Member{}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		mem, err := m.GetMember(ctx, id)
		if err != nil {
			if errors.Is(err, ErrMemberNotFound) {
				continue
			}
			return nil, fmt.Errorf("GetMembers: %w", err)
		}
		out = append(out, mem)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// SetMemberDisplayName records the name a member gave for themselves.
func (m *userManager) SetMemberDisplayName(ctx context.Context, id, displayName string) (Member, error) {
	current, err := m.GetMember(ctx, id)
	if err != nil {
		return Member{}, err
	}
	updated, err := m.dm.UpdateEntity(ctx, id, entitygraph.UpdateEntityRequest{
		Properties: map[string]any{"display_name": displayName},
	})
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Member{}, ErrMemberNotFound
		}
		return Member{}, fmt.Errorf("SetMemberDisplayName: %w", err)
	}
	out := memberFromEntity(updated)
	out.CreatedAt = current.CreatedAt
	return out, nil
}

// ListMembers returns every non-deleted Member, id order.
func (m *userManager) ListMembers(ctx context.Context) ([]Member, error) {
	entities, err := m.dm.ListEntities(ctx, entitygraph.EntityFilter{
		TypeID: memberTypeID,
	})
	if err != nil {
		return nil, fmt.Errorf("ListMembers: %w", err)
	}
	out := make([]Member, 0, len(entities))
	for _, e := range entities {
		out = append(out, memberFromEntity(e))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
