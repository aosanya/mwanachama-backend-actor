// attributes.go — shared Attributes-map uniqueness check for every Create
// that accepts one (CreateActor, CreateGroup, AssignGroup). Required/Range checks are
// pure Go (models.ValidateAttributes, called directly by each *_impl.go);
// Unique needs a database lookup, so it lives here rather than in models,
// which knows nothing about GORM.
package mwanachamaactor

import (
	"context"
	"fmt"

	"gorm.io/datatypes"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

// checkUniqueAttributes reports [ErrDuplicateAttribute] if attrs holds a
// value for a Unique property in properties that already appears on another
// row of table. Does not itself validate Required/Range — callers run
// [models.ValidateAttributes] first.
//
// This is a pre-check, not the only guard: gormstore.Migrate creates a
// matching partial unique index over the same JSON path, so a concurrent
// write that races past this query still fails at the database rather than
// silently duplicating.
func (m *userManager) checkUniqueAttributes(ctx context.Context, table string, properties []models.Property, attrs map[string]any) error {
	for _, p := range properties {
		if !p.Unique {
			continue
		}
		v, ok := attrs[p.Name]
		if !ok || v == nil || v == "" {
			continue
		}
		var count int64
		err := m.db.WithContext(ctx).Table(table).
			Where(datatypes.JSONQuery("attributes").Equals(v, p.Name)).
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
