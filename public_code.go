package mwanachamaactor

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

const (
	publicCodeAlphabet  = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	publicCodeGroupSize = 3
	publicCodeDraws     = 8
	actorCodePrefix     = "M"
	groupCodePrefix     = "G"
)

type publicCodedRole struct {
	role   string
	prefix string
}

var publicCodedRoles = []publicCodedRole{
	{roleActor, actorCodePrefix},
	{roleGroup, groupCodePrefix},
}

var drawCode = drawPublicCode

func drawPublicCode(prefix string, groups int) (string, error) {
	parts := make([]string, 0, groups+1)
	parts = append(parts, prefix)
	limit := big.NewInt(int64(len(publicCodeAlphabet)))
	for g := 0; g < groups; g++ {
		var b strings.Builder
		for c := 0; c < publicCodeGroupSize; c++ {
			n, err := rand.Int(rand.Reader, limit)
			if err != nil {
				return "", fmt.Errorf("draw a code: %w", err)
			}
			b.WriteByte(publicCodeAlphabet[n.Int64()])
		}
		parts = append(parts, b.String())
	}
	return strings.Join(parts, "-"), nil
}

func normalizePublicCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func mintPublicCode(ctx context.Context, tx *gorm.DB, s publicCodeTables, role, prefix string) (string, error) {
	groups, err := codeGroupsIn(ctx, tx, s.format)
	if err != nil {
		return "", err
	}
	table := s.byRole[role]
	for draw := 0; draw < publicCodeDraws; draw++ {
		code, err := drawCode(prefix, groups+draw)
		if err != nil {
			return "", err
		}
		var held int64
		if err := tx.WithContext(ctx).Table(table).Where("public_code = ?", code).Count(&held).Error; err != nil {
			return "", fmt.Errorf("mint a public code: %w", err)
		}
		if held == 0 {
			return code, nil
		}
	}
	return "", fmt.Errorf("mint a public code: %d draws in %s all collided", publicCodeDraws, table)
}

type publicCodeTables struct {
	format string
	byRole map[string]string
}

func (m *userManager) publicCodeTables() publicCodeTables {
	return publicCodeTables{
		format: m.table(roleCodeFormat),
		byRole: map[string]string{roleActor: m.table(roleActor), roleGroup: m.table(roleGroup)},
	}
}

func publicCodeTablesFor(s *spec.Spec) publicCodeTables {
	return publicCodeTables{
		format: tableFor(s, roleCodeFormat),
		byRole: map[string]string{roleActor: tableFor(s, roleActor), roleGroup: tableFor(s, roleGroup)},
	}
}

func codeGroupsIn(ctx context.Context, tx *gorm.DB, formatTable string) (int, error) {
	var rows []map[string]any
	if err := tx.WithContext(ctx).Table(formatTable).
		Where("id = ?", models.CodeFormatID).Limit(1).Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("read the code format: %w", err)
	}
	if len(rows) == 0 {
		return models.DefaultCodeGroups, nil
	}
	n, err := asInt(rows[0]["group_count"])
	if err != nil {
		return 0, fmt.Errorf("read the code format: %w", err)
	}
	if n < 1 {
		return models.DefaultCodeGroups, nil
	}
	return n, nil
}

func (m *userManager) CodeFormat(ctx context.Context) (CodeFormat, error) {
	var out CodeFormat
	q := m.store.Query(ctx, roleCodeFormat).Where("id = ?", models.CodeFormatID)
	errNoFormat := errors.New("no code format")
	if err := m.store.Take(q, roleCodeFormat, &out, errNoFormat); err != nil {
		if errors.Is(err, errNoFormat) {
			return CodeFormat{ID: models.CodeFormatID, GroupCount: models.DefaultCodeGroups}, nil
		}
		return CodeFormat{}, fmt.Errorf("CodeFormat: %w", err)
	}
	return out, nil
}

func (m *userManager) SetCodeFormat(ctx context.Context, groupCount int) (CodeFormat, error) {
	format, err := m.setCodeFormat(ctx, groupCount)
	if errors.Is(err, ErrDuplicateID) {
		format, err = m.setCodeFormat(ctx, groupCount)
	}
	if err != nil && !errors.Is(err, ErrInvalidCodeFormat) {
		return CodeFormat{}, fmt.Errorf("SetCodeFormat: %w", err)
	}
	return format, err
}

func (m *userManager) setCodeFormat(ctx context.Context, groupCount int) (CodeFormat, error) {
	if groupCount < 1 || groupCount > models.MaxCodeGroups {
		return CodeFormat{}, fmt.Errorf("%w: a code has between 1 and %d groups, not %d",
			ErrInvalidCodeFormat, models.MaxCodeGroups, groupCount)
	}
	format := CodeFormat{ID: models.CodeFormatID, GroupCount: groupCount, LastUpdated: NowRFC3339()}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated := tx.Table(m.table(roleCodeFormat)).Where("id = ?", format.ID).
			Updates(map[string]any{"group_count": format.GroupCount, "last_updated": format.LastUpdated})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected > 0 {
			return nil
		}
		row, err := encode(m.object(roleCodeFormat), format)
		if err != nil {
			return err
		}
		if err := tx.Table(m.table(roleCodeFormat)).Create(row).Error; err != nil {
			return classifyDuplicateID(err)
		}
		return nil
	})
	if err != nil {
		return CodeFormat{}, err
	}
	return format, nil
}
