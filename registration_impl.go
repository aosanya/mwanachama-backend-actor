// registration_impl.go — the registered_at edge implementation for
// [userManager]. Ported from mwanachama-backend-api-gateway's
// internal/store/postgres/member_registration_store.go and
// internal/store/memory/member_store.go's registration half.
//
// DSN-1698 decision 4: Registration is a Member -> Group relationship edge
// (RelRegisteredAt), not a table. Decision 8: "one home group per member" is
// dropped entirely — Register below does NOT clear any other registration's
// IsHome flag, unlike the gateway's pre-cutover Register.
//
// entitygraph.DataManager has no UpdateRelationship method (see
// mwanachama-backend-shared/entitygraph/entitygraph.go) — a re-registration
// is therefore implemented as delete-then-recreate, preserving the original
// JoinedAt (DEV-1319: a join date describes the first join).
package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/aosanya/mwanachama-backend-shared/entitygraph"
)

// findRegistrationEdge returns the registered_at edge from memberID to
// groupID, if one exists.
func (m *userManager) findRegistrationEdge(ctx context.Context, memberID, groupID string) (entitygraph.Relationship, bool, error) {
	edges, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		FromID: memberID,
		ToID:   groupID,
		Name:   RelRegisteredAt,
	})
	if err != nil {
		return entitygraph.Relationship{}, false, err
	}
	if len(edges) == 0 {
		return entitygraph.Relationship{}, false, nil
	}
	return edges[0], true, nil
}

// Register enrols a member at a group (upsert on (memberID, groupID)). See
// the package doc above for why re-registration is delete-then-recreate, and
// the file doc for why IsHome carries no exclusivity.
func (m *userManager) Register(ctx context.Context, r Registration) (Registration, error) {
	existing, found, err := m.findRegistrationEdge(ctx, r.MemberID, r.GroupID)
	if err != nil {
		return Registration{}, fmt.Errorf("Register: %w", err)
	}
	if found {
		if err := m.dm.DeleteRelationship(ctx, existing.ID); err != nil && !errors.Is(err, entitygraph.ErrRelationshipNotFound) {
			return Registration{}, fmt.Errorf("Register: replace: %w", err)
		}
		if r.JoinedAt == "" {
			r.JoinedAt = entitygraph.StringProp(existing.Properties, "joined_at")
		}
	}
	if r.JoinedAt == "" {
		r.JoinedAt = nowRFC3339()
	}

	_, err = m.dm.CreateRelationship(ctx, entitygraph.CreateRelationshipRequest{
		Name:   RelRegisteredAt,
		FromID: r.MemberID,
		ToID:   r.GroupID,
		Properties: map[string]any{
			"is_home":   r.IsHome,
			"joined_at": r.JoinedAt,
		},
	})
	if err != nil {
		if errors.Is(err, entitygraph.ErrEntityNotFound) {
			return Registration{}, ErrMemberNotFound
		}
		return Registration{}, fmt.Errorf("Register: %w", err)
	}
	return r, nil
}

// Deregister removes a member's registered_at edge to a group. Returns
// found=false and no error when no such edge exists (idempotent no-op).
//
// No act-log row is written here — see UserManager.Deregister's doc. The
// caller (the gateway adapter) is responsible for composing and writing
// whatever record its own domain wants of the removal, using the returned
// Registration (which carries IsHome — the one fact about a departure that
// is unrecoverable once the edge is gone).
func (m *userManager) Deregister(ctx context.Context, memberID, groupID string) (Registration, bool, error) {
	edge, found, err := m.findRegistrationEdge(ctx, memberID, groupID)
	if err != nil {
		return Registration{}, false, fmt.Errorf("Deregister: %w", err)
	}
	if !found {
		return Registration{}, false, nil
	}
	reg := registrationFromEdge(edge)
	if err := m.dm.DeleteRelationship(ctx, edge.ID); err != nil {
		if errors.Is(err, entitygraph.ErrRelationshipNotFound) {
			return Registration{}, false, nil
		}
		return Registration{}, false, fmt.Errorf("Deregister: %w", err)
	}
	return reg, true, nil
}

// ListGroupsForMember returns every registration a member holds, joined_at-
// then-group-id order (matching the gateway's ORDER BY joined_at, chapter_id).
func (m *userManager) ListGroupsForMember(ctx context.Context, memberID string) ([]Registration, error) {
	edges, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		FromID: memberID,
		Name:   RelRegisteredAt,
	})
	if err != nil {
		return nil, fmt.Errorf("ListGroupsForMember: %w", err)
	}
	out := make([]Registration, 0, len(edges))
	for _, e := range edges {
		out = append(out, registrationFromEdge(e))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].JoinedAt != out[j].JoinedAt {
			return out[i].JoinedAt < out[j].JoinedAt
		}
		return out[i].GroupID < out[j].GroupID
	})
	return out, nil
}

// ListMembersForGroup returns every member registered at a group, joined_at-
// then-member-id order (matching the gateway's ORDER BY joined_at, member_id).
func (m *userManager) ListMembersForGroup(ctx context.Context, groupID string) ([]Registration, error) {
	edges, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		ToID: groupID,
		Name: RelRegisteredAt,
	})
	if err != nil {
		return nil, fmt.Errorf("ListMembersForGroup: %w", err)
	}
	out := make([]Registration, 0, len(edges))
	for _, e := range edges {
		out = append(out, registrationFromEdge(e))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].JoinedAt != out[j].JoinedAt {
			return out[i].JoinedAt < out[j].JoinedAt
		}
		return out[i].MemberID < out[j].MemberID
	})
	return out, nil
}

// HomeCounts tallies members by home group. `is_home` only, not merely
// "registered at" — a member registered at three groups may call more than
// one of them home now that decision 8 drops exclusivity, but the count
// still means "members who marked this their home", not "members registered
// here", matching the gateway's own HomeCounts contract.
func (m *userManager) HomeCounts(ctx context.Context) (map[string]int, error) {
	edges, err := m.dm.ListRelationships(ctx, entitygraph.RelationshipFilter{
		Name: RelRegisteredAt,
	})
	if err != nil {
		return nil, fmt.Errorf("HomeCounts: %w", err)
	}
	out := map[string]int{}
	for _, e := range edges {
		if entitygraph.BoolProp(e.Properties, "is_home") {
			out[e.ToID]++
		}
	}
	return out, nil
}

func registrationFromEdge(e entitygraph.Relationship) Registration {
	return Registration{
		MemberID: e.FromID,
		GroupID:  e.ToID,
		IsHome:   entitygraph.BoolProp(e.Properties, "is_home"),
		JoinedAt: entitygraph.StringProp(e.Properties, "joined_at"),
	}
}
