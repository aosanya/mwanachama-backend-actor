// registration_impl.go — the registered_at relationship implementation for
// [userManager]. Ported from mwanachama-backend-api-gateway's
// internal/store/postgres/member_registration_store.go and
// internal/store/memory/member_store.go's registration half.
//
// DSN-1698 decision 4: ActorGroupAssignment is an Actor -> Group
// relationship, now a real row in m.tables.ActorGroupAssignments keyed by
// the composite primary key (actor_id, group_id) — see gormstore's
// ActorGroupAssignmentRow doc.
//
// **No IsHome/JoinedAt columns, 2026-09-04** — see models.ActorGroupAssignment's
// doc. CreatedAt now plays JoinedAt's old role (the moment this assignment
// first existed) and AssignGroup preserves it across a re-assign the same
// way it used to preserve JoinedAt. Decision 8's "no exclusivity" is
// unaffected either way: nothing here enforces at-most-one-home regardless
// of where (or whether) a caller keeps that flag.
//
// AssignGroup keeps the same read-then-write shape entitygraph forced on it
// (no UpdateRelationship there), rather than switching to a GORM ON
// CONFLICT upsert now that the storage is relational — same tested
// behavior, no new concurrency semantics.
package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-actor/gormstore"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

// findAssignmentRow returns the assignment row for (actorID, groupID), if
// one exists.
func (m *userManager) findAssignmentRow(ctx context.Context, actorID, groupID string) (gormstore.ActorGroupAssignmentRow, bool, error) {
	var row gormstore.ActorGroupAssignmentRow
	err := m.db.WithContext(ctx).Table(m.tables.ActorGroupAssignments).
		Where("actor_id = ? AND group_id = ?", actorID, groupID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gormstore.ActorGroupAssignmentRow{}, false, nil
	}
	if err != nil {
		return gormstore.ActorGroupAssignmentRow{}, false, err
	}
	return row, true, nil
}

// AssignGroup enrols an actor at a group (upsert on (actorID, groupID)).
// CreatedAt is set once, on the first assignment of this pair, and
// preserved across every later re-assign — mirrors the old JoinedAt
// preserve-on-update behavior. LastUpdated is stamped on every call.
// Attributes is validated against
// [models.DefaultActorGroupAssignmentProperties] first — mirrors
// CreateActor/CreateGroup's validation, though a Unique-property collision
// is only checked when a new row is being created (see the
// checkUniqueAttributes call below), not on an update of an existing
// assignment's own row.
func (m *userManager) AssignGroup(ctx context.Context, r models.ActorGroupAssignment) (models.ActorGroupAssignment, error) {
	properties := models.DefaultActorGroupAssignmentProperties()
	if err := models.ValidateAttributes(properties, r.Attributes); err != nil {
		return models.ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
	}

	existing, found, err := m.findAssignmentRow(ctx, r.ActorID, r.GroupID)
	if err != nil {
		return models.ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
	}
	if found {
		r.CreatedAt = existing.CreatedAt
	} else if r.CreatedAt == "" {
		r.CreatedAt = models.NowRFC3339()
	}
	r.LastUpdated = models.NowRFC3339()

	row := gormstore.ActorGroupAssignmentToRow(r)
	if found {
		err = m.db.WithContext(ctx).Table(m.tables.ActorGroupAssignments).
			Where("actor_id = ? AND group_id = ?", r.ActorID, r.GroupID).
			Updates(map[string]any{"updated_at": row.UpdatedAt, "attributes": row.Attributes}).Error
	} else {
		// An assignment references an actor and a group by id, but this
		// table carries no FK to either (see gormstore's
		// ActorGroupAssignmentRow doc) — check the actor exists explicitly,
		// mirroring entitygraph's CreateEntity-backed ErrEntityNotFound
		// behavior the tests already exercise.
		var actorExists int64
		if cerr := m.db.WithContext(ctx).Table(m.tables.Actors).Where("id = ?", r.ActorID).Count(&actorExists).Error; cerr != nil {
			return models.ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", cerr)
		}
		if actorExists == 0 {
			return models.ActorGroupAssignment{}, ErrActorNotFound
		}
		if err := m.checkUniqueAttributes(ctx, m.tables.ActorGroupAssignments, properties, r.Attributes); err != nil {
			return models.ActorGroupAssignment{}, err
		}
		err = m.db.WithContext(ctx).Table(m.tables.ActorGroupAssignments).Create(&row).Error
	}
	if err != nil {
		return models.ActorGroupAssignment{}, fmt.Errorf("AssignGroup: %w", err)
	}
	return r, nil
}

// Deregister removes an actor's registered_at row for a group. Returns
// found=false and no error when no such row exists (idempotent no-op).
//
// No act-log row is written here — see UserManager.Deregister's doc. The
// caller (the gateway adapter) is responsible for composing and writing
// whatever record its own domain wants of the removal, using the returned
// ActorGroupAssignment.
func (m *userManager) Deregister(ctx context.Context, actorID, groupID string) (models.ActorGroupAssignment, bool, error) {
	row, found, err := m.findAssignmentRow(ctx, actorID, groupID)
	if err != nil {
		return models.ActorGroupAssignment{}, false, fmt.Errorf("Deregister: %w", err)
	}
	if !found {
		return models.ActorGroupAssignment{}, false, nil
	}
	reg := gormstore.ActorGroupAssignmentFromRow(row)
	err = m.db.WithContext(ctx).Table(m.tables.ActorGroupAssignments).
		Where("actor_id = ? AND group_id = ?", actorID, groupID).Delete(&gormstore.ActorGroupAssignmentRow{}).Error
	if err != nil {
		return models.ActorGroupAssignment{}, false, fmt.Errorf("Deregister: %w", err)
	}
	return reg, true, nil
}

// ListGroupsForActor returns every registration an actor holds, created_at-
// then-group-id order (matching the gateway's ORDER BY joined_at,
// chapter_id — created_at now plays that role, see the package doc).
func (m *userManager) ListGroupsForActor(ctx context.Context, actorID string) ([]models.ActorGroupAssignment, error) {
	var rows []gormstore.ActorGroupAssignmentRow
	err := m.db.WithContext(ctx).Table(m.tables.ActorGroupAssignments).Where("actor_id = ?", actorID).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ListGroupsForActor: %w", err)
	}
	out := make([]models.ActorGroupAssignment, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.ActorGroupAssignmentFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].GroupID < out[j].GroupID
	})
	return out, nil
}

// ListActorsForGroup returns every actor registered at a group, created_at-
// then-actor-id order (matching the gateway's ORDER BY joined_at,
// member_id — created_at now plays that role).
func (m *userManager) ListActorsForGroup(ctx context.Context, groupID string) ([]models.ActorGroupAssignment, error) {
	var rows []gormstore.ActorGroupAssignmentRow
	err := m.db.WithContext(ctx).Table(m.tables.ActorGroupAssignments).Where("group_id = ?", groupID).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ListActorsForGroup: %w", err)
	}
	out := make([]models.ActorGroupAssignment, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.ActorGroupAssignmentFromRow(r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].ActorID < out[j].ActorID
	})
	return out, nil
}

// HomeCounts tallies actors by home group. Since "home" is no longer a
// dedicated column (see the package doc), this reads the caller-declared
// `Attributes["is_home"]` JSON path rather than a boolean column — a group
// is counted for an actor only when that actor's own assignment row set
// is_home=true in Attributes; this package itself attaches no meaning to
// the key otherwise, and a deployment that never sets it gets an empty map
// back, not an error. Groups with nobody are absent rather than zero.
func (m *userManager) HomeCounts(ctx context.Context) (map[string]int, error) {
	var rows []gormstore.ActorGroupAssignmentRow
	err := m.db.WithContext(ctx).Table(m.tables.ActorGroupAssignments).
		Where(datatypes.JSONQuery("attributes").Equals(true, "is_home")).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("HomeCounts: %w", err)
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.GroupID]++
	}
	return out, nil
}
