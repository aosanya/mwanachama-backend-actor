package mwanachamaactor

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/aosanya/mwanachama-backend-shared/plaintext"
)

func checkName(refusal error, label, name string) error {
	if problem := plaintext.Problem(label, name); problem != "" {
		return fmt.Errorf("%w: %s", refusal, problem)
	}
	return nil
}

const maxNameCharacters = 200

func checkRequiredName(refusal error, label, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: a %s is required", refusal, label)
	}
	if utf8.RuneCountInString(name) > maxNameCharacters {
		return fmt.Errorf("%w: a %s may be at most %d characters", refusal, label, maxNameCharacters)
	}
	return checkName(refusal, label, name)
}

func checkGroupName(name string) error {
	return checkRequiredName(ErrInvalidGroup, "group name", name)
}

func checkHierarchyName(name string) error {
	return checkRequiredName(ErrInvalidHierarchy, "hierarchy name", name)
}

func checkActorDisplayName(name string) error {
	if name == "" {
		return nil
	}
	return checkRequiredName(ErrInvalidActor, "display name", name)
}

func checkActorPersonNames(firstName, lastName string) error {
	for _, field := range []struct{ label, name string }{
		{"first name", firstName},
		{"last name", lastName},
	} {
		if err := checkName(ErrInvalidActor, field.label, field.name); err != nil {
			return err
		}
	}
	return nil
}

func checkGroupTypeNames(name, singular, plural string) error {
	for _, field := range []struct{ label, name string }{
		{"group type name", name},
		{"group type singular", singular},
		{"group type plural", plural},
	} {
		if err := checkName(ErrInvalidGroupType, field.label, field.name); err != nil {
			return err
		}
	}
	return nil
}
