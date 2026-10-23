package mwanachamaactor

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/datatypes"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) CorrectMyAttributes(ctx context.Context, actorID string, correction models.AttributeCorrection) (Actor, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return Actor{}, ErrActorNotFound
	}
	return m.setAttributes(ctx, actorID, actorID, models.AttributeSetBySelf, correction.Attributes)
}

func (m *userManager) SetActorAttributes(ctx context.Context, actorID, staffID string, correction models.AttributeCorrection) (Actor, error) {
	staffID = strings.TrimSpace(staffID)
	if staffID == "" {
		return Actor{}, fmt.Errorf("%w: a staff edit needs the editor", ErrInvalidActor)
	}
	return m.setAttributes(ctx, strings.TrimSpace(actorID), staffID, models.AttributeSetByStaff, correction.Attributes)
}

func (m *userManager) setAttributes(ctx context.Context, actorID, editorID, setBy string, changes map[string]any) (Actor, error) {
	held, err := m.GetActor(ctx, actorID)
	if err != nil {
		return Actor{}, err
	}
	if len(changes) == 0 {
		return Actor{}, fmt.Errorf("%w: no attribute to set", ErrInvalidActor)
	}
	properties := models.DefaultActorProperties()
	declared := map[string]bool{}
	for _, p := range properties {
		declared[p.Name] = true
	}
	merged := map[string]any{}
	for k, v := range held.Attributes {
		merged[k] = v
	}
	sources := map[string]models.AttributeSource{}
	for k, v := range held.AttributeSources {
		sources[k] = v
	}
	now := NowRFC3339()
	for name, value := range changes {
		if !declared[name] {
			return Actor{}, fmt.Errorf("%w: %q is not a declared attribute", ErrInvalidActor, name)
		}
		if value == nil || value == "" {
			delete(merged, name)
		} else {
			merged[name] = value
		}
		sources[name] = models.AttributeSource{SetBy: setBy, SetByID: editorID, SetAt: now}
	}
	if err := models.ValidateAttributes(properties, merged); err != nil {
		return Actor{}, fmt.Errorf("%w: %v", ErrInvalidActor, err)
	}
	if err := m.checkUniqueAttributesBesides(ctx, actorID, properties, changes); err != nil {
		return Actor{}, err
	}
	held.Attributes = merged
	held.AttributeSources = sources
	held.LastUpdated = now
	row, err := encode(m.object(roleActor), held)
	if err != nil {
		return Actor{}, err
	}
	update := map[string]any{
		"attributes":        row["attributes"],
		"attribute_sources": row["attribute_sources"],
		"last_updated":      row["last_updated"],
	}
	if err := m.store.Query(ctx, roleActor).Where("id = ?", actorID).Updates(update).Error; err != nil {
		return Actor{}, fmt.Errorf("setAttributes: %w", classifyDuplicateID(err))
	}
	return held, nil
}

func (m *userManager) checkUniqueAttributesBesides(ctx context.Context, actorID string, properties []models.Property, changes map[string]any) error {
	for _, p := range properties {
		v, ok := changes[p.Name]
		if !p.Unique || !ok || v == nil || v == "" {
			continue
		}
		var count int64
		err := m.db.WithContext(ctx).Table(m.table(roleActor)).
			Where(datatypes.JSONQuery("attributes").Equals(v, p.Name)).
			Where("id <> ?", actorID).
			Count(&count).Error
		if err != nil {
			return fmt.Errorf("checkUniqueAttributes: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("%w: %q = %v", ErrDuplicateAttribute, p.Name, v)
		}
	}
	return nil
}
