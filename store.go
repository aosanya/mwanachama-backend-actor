package mwanachamaactor

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

const (
	roleActor                = "actor"
	roleGroup                = "group"
	roleActorGroupAssignment = "actor_group_assignment"
	roleRoleKind             = "role_kind"
	roleActorRoleAssignment  = "actor_role_assignment"
	roleHierarchy            = "hierarchy"
	roleLevel                = "level"
	roleGroupType            = "group_type"
	roleGroupJoinRequest     = "group_join_request"
	roleJoinDecision         = "join_decision"
	roleGroupRemoval         = "group_removal"
	roleDelegation           = "delegation"
	roleGroupInvitation      = "group_invitation"
	roleCodeSequence         = "code_sequence"
	roleCodeFormat           = "code_format"
	roleGroupNameTranslation = "group_name_translation"
	roleGroupPopulation      = "group_population"
)

func carriers() map[string]any {
	return map[string]any{
		roleActor:                models.Actor{},
		roleGroup:                models.Group{},
		roleActorGroupAssignment: models.ActorGroupAssignment{},
		roleRoleKind:             models.RoleKind{},
		roleActorRoleAssignment:  models.ActorRoleAssignment{},
		roleHierarchy:            models.Hierarchy{},
		roleLevel:                models.Level{},
		roleGroupType:            models.GroupType{},
		roleGroupJoinRequest:     models.GroupJoinRequest{},
		roleJoinDecision:         models.JoinDecision{},
		roleGroupRemoval:         models.GroupRemoval{},
		roleDelegation:           models.Delegation{},
		roleGroupInvitation:      models.GroupInvitation{},
		roleCodeSequence:         models.CodeSequence{},
		roleCodeFormat:           models.CodeFormat{},
		roleGroupNameTranslation: models.GroupNameTranslation{},
		roleGroupPopulation:      models.GroupPopulation{},
	}
}

type store = specstore.Store

func newStore(db *gorm.DB, s *spec.Spec) (*store, error) {
	return specstore.New(db, s, carriers())
}

func newSpecID() string { return specstore.NewID() }

func columnName(field string) string { return specstore.ColumnName(field) }

func encode(o spec.Object, v any) (map[string]any, error) { return specstore.Encode(o, v) }

func decode(o spec.Object, row map[string]any, out any) error {
	return specstore.Decode(o, row, out)
}

const codeSequenceAttempts = 8

type codedRole struct {
	role       string
	entityType string
	prefix     string
}

var codedRoles = []codedRole{
	{roleActor, "actor", "AC"},
	{roleGroup, "group", "G"},
	{roleHierarchy, "hierarchy", "H"},
	{roleLevel, "level", "L"},
	{roleRoleKind, "role_kind", "RK"},
	{roleGroupType, "grouptype", "GT"},
}

func nextCode(ctx context.Context, tx *gorm.DB, table, entityType, prefix string) (string, error) {
	for attempt := 0; attempt < codeSequenceAttempts; attempt++ {
		var rows []map[string]any
		if err := tx.WithContext(ctx).Table(table).
			Where("entity_type = ?", entityType).Limit(1).Find(&rows).Error; err != nil {
			return "", fmt.Errorf("nextCode: %w", err)
		}

		if len(rows) == 0 {
			seed := map[string]any{"entity_type": entityType, "next_number": 2}
			if err := tx.WithContext(ctx).Table(table).Create(seed).Error; err != nil {
				return "", fmt.Errorf("nextCode: %w", err)
			}
			return fmt.Sprintf("%s-%d", prefix, 1), nil
		}

		n, err := asInt(rows[0]["next_number"])
		if err != nil {
			return "", fmt.Errorf("nextCode: %s: %w", entityType, err)
		}

		claimed := tx.WithContext(ctx).Table(table).
			Where("entity_type = ? AND next_number = ?", entityType, n).
			Update("next_number", n+1)
		if claimed.Error != nil {
			return "", fmt.Errorf("nextCode: %w", claimed.Error)
		}
		if claimed.RowsAffected == 1 {
			return fmt.Sprintf("%s-%d", prefix, n), nil
		}
	}
	return "", fmt.Errorf("nextCode: %s: the counter moved under %d attempts", entityType, codeSequenceAttempts)
}

func asInt(v any) (int, error) {
	switch n := v.(type) {
	case int64:
		return int(n), nil
	case int:
		return n, nil
	case float64:
		return int(n), nil
	default:
		return 0, fmt.Errorf("cannot read %T as a number", v)
	}
}
