package mwanachamaactor

import (
	"context"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) ScreenNameByCode(ctx context.Context, code string) (models.ScreenName, error) {
	actor, err := m.getActorByCode(ctx, code)
	if err != nil {
		return models.ScreenName{}, err
	}
	return models.ScreenName{DisplayName: actor.DisplayName}, nil
}
