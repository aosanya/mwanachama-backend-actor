// actor_impl.go — Actor CRUD implementation for [userManager]. Ported
// from mwanachama-backend-api-gateway's internal/store/postgres/member_store.go
// and internal/store/memory/member_store.go (the identity half; the
// registration half is in registration_impl.go, mirroring the gateway's own
// member_store.go / member_registration_store.go split).
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

// CreateActor creates an Actor entity. Attributes is validated against
// [models.DefaultActorProperties] first — a Required property missing or
// blank, a value of the wrong Range, or a Unique property already held by
// another actor all fail the call before any row is written.
func (m *userManager) CreateActor(ctx context.Context, act Actor) (Actor, error) {
	properties := models.DefaultActorProperties()
	if err := models.ValidateAttributes(properties, act.Attributes); err != nil {
		return Actor{}, fmt.Errorf("%w: %v", ErrInvalidActor, err)
	}
	if err := m.checkUniqueAttributes(ctx, m.tables.Actors, properties, act.Attributes); err != nil {
		return Actor{}, err
	}
	if act.CreatedAt == "" {
		act.CreatedAt = NowRFC3339()
	}
	row := gormstore.ActorToRow(act)
	if err := m.db.WithContext(ctx).Table(m.tables.Actors).Create(&row).Error; err != nil {
		return Actor{}, fmt.Errorf("CreateActor: %w", err)
	}
	return gormstore.ActorFromRow(row), nil
}

// GetActor reads a single Actor entity.
func (m *userManager) GetActor(ctx context.Context, id string) (Actor, error) {
	var row gormstore.ActorRow
	err := m.db.WithContext(ctx).Table(m.tables.Actors).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Actor{}, ErrActorNotFound
		}
		return Actor{}, fmt.Errorf("GetActor: %w", err)
	}
	return gormstore.ActorFromRow(row), nil
}

// GetActors returns the Actors for the given ids, sorted by id. Ids with
// no actor are skipped rather than erroring.
func (m *userManager) GetActors(ctx context.Context, ids []string) ([]Actor, error) {
	out := []Actor{}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		act, err := m.GetActor(ctx, id)
		if err != nil {
			if errors.Is(err, ErrActorNotFound) {
				continue
			}
			return nil, fmt.Errorf("GetActors: %w", err)
		}
		out = append(out, act)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// SetActorDisplayName records the name an actor gave for themselves.
func (m *userManager) SetActorDisplayName(ctx context.Context, id, displayName string) (Actor, error) {
	current, err := m.GetActor(ctx, id)
	if err != nil {
		return Actor{}, err
	}
	now := NowRFC3339()
	err = m.db.WithContext(ctx).Table(m.tables.Actors).Where("id = ?", id).
		Updates(map[string]any{"display_name": displayName, "updated_at": now}).Error
	if err != nil {
		return Actor{}, fmt.Errorf("SetActorDisplayName: %w", err)
	}
	current.DisplayName = displayName
	current.LastUpdated = now
	return current, nil
}

// ListActors returns every non-deleted Actor, id order.
func (m *userManager) ListActors(ctx context.Context) ([]Actor, error) {
	var rows []gormstore.ActorRow
	if err := m.db.WithContext(ctx).Table(m.tables.Actors).Where("deleted = ?", false).Order("id").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListActors: %w", err)
	}
	out := make([]Actor, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.ActorFromRow(r))
	}
	return out, nil
}
