package mwanachamaactor

import (
	"context"
	"fmt"
	"strings"

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

func decode(o spec.Object, row map[string]any, out any) error {
	return specstore.Decode(o, row, out)
}

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

func inRun(q *gorm.DB, runID string) *gorm.DB {
	if runID == "" {
		return q
	}
	return q.Where(spec.RunIDColumn+" = ?", runID)
}

func nextCode(ctx context.Context, tx *gorm.DB, table, runID, entityType, prefix string) (string, error) {
	columns := []string{"entity_type", "next_number"}
	values := []any{entityType, 2}
	conflict := []string{"entity_type"}
	if runID != "" {
		columns = append(columns, spec.RunIDColumn)
		values = append(values, runID)
		conflict = []string{spec.RunIDColumn, "entity_type"}
	}
	quoted := tx.Statement.Quote(table)
	statement := fmt.Sprintf(
		"INSERT INTO %s AS counter (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET next_number = counter.next_number + 1 RETURNING next_number",
		quoted, strings.Join(columns, ", "), strings.TrimSuffix(strings.Repeat("?, ", len(values)), ", "), strings.Join(conflict, ", "))

	var advanced int
	if err := tx.WithContext(ctx).Raw(statement, values...).Scan(&advanced).Error; err != nil {
		return "", fmt.Errorf("nextCode: %s: %w", entityType, err)
	}
	return fmt.Sprintf("%s-%d", prefix, advanced-1), nil
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
