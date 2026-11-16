package mwanachamaactor

import (
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/plaintext"
)

func checkName(refusal error, label, name string) error {
	if problem := plaintext.Problem(label, name); problem != "" {
		return fmt.Errorf("%w: %s", refusal, problem)
	}
	return nil
}

func checkActorNames(displayName, firstName, lastName string) error {
	for _, field := range []struct{ label, name string }{
		{"display name", displayName},
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
