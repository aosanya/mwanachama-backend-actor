package mwanachamaactor

import (
	"context"
	"fmt"

	"gorm.io/datatypes"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) checkUniqueAttributes(ctx context.Context, role string, properties []models.Property, attrs map[string]any) error {
	for _, p := range properties {
		if !p.Unique {
			continue
		}
		v, ok := attrs[p.Name]
		if !ok || v == nil || v == "" {
			continue
		}
		var count int64
		err := m.from(m.db.WithContext(ctx), role).
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
