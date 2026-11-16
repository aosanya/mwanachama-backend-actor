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

func checkRequiredName(refusal error, label, name string) (string, error) {
	return checkRequiredNameWorded(refusal, label, label, name)
}

func checkRequiredNameWorded(refusal error, wording, label, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: a %s is required", refusal, wording)
	}
	if utf8.RuneCountInString(name) > maxNameCharacters {
		return "", fmt.Errorf("%w: a %s may be at most %d characters", refusal, wording, maxNameCharacters)
	}
	return name, checkName(refusal, label, name)
}

func checkGroupName(name string) (string, error) {
	return checkRequiredName(ErrInvalidGroup, "group name", name)
}

func checkGroupEditName(name string) (string, error) {
	return checkRequiredNameWorded(ErrInvalidGroup, "name", "group name", name)
}

func checkHierarchyName(name string) (string, error) {
	return checkRequiredName(ErrInvalidHierarchy, "hierarchy name", name)
}

func checkActorDisplayName(name string) (string, error) {
	if name == "" {
		return "", nil
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
