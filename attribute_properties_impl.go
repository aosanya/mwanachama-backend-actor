package mwanachamaactor

import (
	"context"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) AttributeProperties(ctx context.Context) ([]models.Property, error) {
	return models.DefaultActorProperties(), nil
}
