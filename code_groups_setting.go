package mwanachamaactor

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

const codeGroupsSetting = "code_groups"

func settledCodeGroups(db *gorm.DB, s *spec.Spec) (int, error) {
	if s == nil {
		return models.DefaultCodeGroups, nil
	}
	if _, declared := s.Setting(codeGroupsSetting); !declared {
		return models.DefaultCodeGroups, nil
	}
	effective, err := spec.ResolveStored(db, s, codeGroupsSetting)
	if err != nil {
		return 0, fmt.Errorf("resolve %s: %w", codeGroupsSetting, err)
	}
	return models.ParseCodeGroups(effective.Value)
}
