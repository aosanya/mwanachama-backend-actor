package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) CreateActor(ctx context.Context, act Actor) (Actor, error) {
	properties := models.DefaultActorProperties()
	if err := models.ValidateAttributes(properties, act.Attributes); err != nil {
		return Actor{}, fmt.Errorf("%w: %v", ErrInvalidActor, err)
	}
	if err := m.checkUniqueAttributes(ctx, m.table(roleActor), properties, act.Attributes); err != nil {
		return Actor{}, err
	}

	act.ID = newSpecID()
	return m.insertActor(ctx, act)
}

func (m *userManager) EnsureActor(ctx context.Context, id string) (Actor, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Actor{}, fmt.Errorf("%w: an actor needs an id", ErrInvalidActor)
	}
	held, err := m.GetActor(ctx, id)
	if err == nil {
		return held, nil
	}
	if !errors.Is(err, ErrActorNotFound) {
		return Actor{}, fmt.Errorf("EnsureActor: %w", err)
	}
	created, err := m.insertActor(ctx, Actor{ID: id})
	if err == nil {
		return created, nil
	}
	if !errors.Is(err, ErrDuplicateID) {
		return Actor{}, fmt.Errorf("EnsureActor: %w", err)
	}
	held, err = m.GetActor(ctx, id)
	if err != nil {
		return Actor{}, fmt.Errorf("EnsureActor: %w", err)
	}
	return held, nil
}

func (m *userManager) insertActor(ctx context.Context, act Actor) (Actor, error) {
	act.CreatedAt = NowRFC3339()
	act.LastUpdated = act.CreatedAt
	act.Deleted = false
	act.FormsWeekSplit = true

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		code, err := nextCode(ctx, tx, m.table(roleCodeSequence), "actor", "AC")
		if err != nil {
			return err
		}
		act.Code = code
		act.PublicCode, err = mintPublicCode(ctx, tx, m.publicCodeTables(), roleActor, actorCodePrefix)
		if err != nil {
			return err
		}
		row, err := encode(m.object(roleActor), act)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Table(m.table(roleActor)).Create(row).Error; err != nil {
			return classifyDuplicateID(err)
		}
		return nil
	})
	if err != nil {
		return Actor{}, fmt.Errorf("CreateActor: %w", err)
	}
	return act, nil
}

func classifyDuplicateID(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if isAttributeIndex(pgErr.ConstraintName) {
			return ErrDuplicateAttribute
		}
		return ErrDuplicateID
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		if isAttributeIndex(err.Error()) {
			return ErrDuplicateAttribute
		}
		return ErrDuplicateID
	}
	return err
}

func isAttributeIndex(s string) bool {
	return strings.Contains(s, "_attributes_")
}

func (m *userManager) GetActorByPhone(ctx context.Context, phone string) (Actor, error) {
	var out Actor
	q := m.store.Query(ctx, roleActor).
		Where(datatypes.JSONQuery("attributes").Equals(phone, "phone")).
		Where("deleted = ?", false)
	if err := m.store.Take(q, roleActor, &out, ErrActorNotFound); err != nil {
		if errors.Is(err, ErrActorNotFound) {
			return Actor{}, ErrActorNotFound
		}
		return Actor{}, fmt.Errorf("GetActorByPhone: %w", err)
	}
	return out, nil
}

func (m *userManager) GetActor(ctx context.Context, id string) (Actor, error) {
	var out Actor
	q := m.store.Query(ctx, roleActor).Where("id = ?", id).Where("deleted = ?", false)
	if err := m.store.Take(q, roleActor, &out, ErrActorNotFound); err != nil {
		if errors.Is(err, ErrActorNotFound) {
			return Actor{}, ErrActorNotFound
		}
		return Actor{}, fmt.Errorf("GetActor: %w", err)
	}
	return out, nil
}

func (m *userManager) getActorByCode(ctx context.Context, code string) (Actor, error) {
	code = normalizePublicCode(code)
	if code == "" {
		return Actor{}, ErrActorNotFound
	}
	var out Actor
	q := m.store.Query(ctx, roleActor).Where("public_code = ?", code).Where("deleted = ?", false)
	if err := m.store.Take(q, roleActor, &out, ErrActorNotFound); err != nil {
		if errors.Is(err, ErrActorNotFound) {
			return Actor{}, ErrActorNotFound
		}
		return Actor{}, fmt.Errorf("getActorByCode: %w", err)
	}
	return out, nil
}

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

func (m *userManager) SetActorDisplayName(ctx context.Context, id, displayName string) (Actor, error) {
	current, err := m.GetActor(ctx, id)
	if err != nil {
		return Actor{}, err
	}
	now := NowRFC3339()
	err = m.store.Query(ctx, roleActor).Where("id = ?", id).
		Updates(map[string]any{"display_name": displayName, "last_updated": now}).Error
	if err != nil {
		return Actor{}, fmt.Errorf("SetActorDisplayName: %w", err)
	}
	current.DisplayName = displayName
	current.LastUpdated = now
	return current, nil
}

func (m *userManager) SetActorName(ctx context.Context, id, firstName, lastName string) (Actor, error) {
	current, err := m.GetActor(ctx, id)
	if err != nil {
		return Actor{}, err
	}
	now := NowRFC3339()
	err = m.store.Query(ctx, roleActor).Where("id = ?", id).
		Updates(map[string]any{"first_name": firstName, "last_name": lastName, "last_updated": now}).Error
	if err != nil {
		return Actor{}, fmt.Errorf("SetActorName: %w", err)
	}
	current.FirstName = firstName
	current.LastName = lastName
	current.LastUpdated = now
	return current, nil
}

func (m *userManager) ListActors(ctx context.Context) ([]Actor, error) {
	q := m.store.Query(ctx, roleActor).Where("deleted = ?", false).Order("id")
	out, err := specstore.List[Actor](m.store, q, roleActor)
	if err != nil {
		return nil, fmt.Errorf("ListActors: %w", err)
	}
	return out, nil
}
