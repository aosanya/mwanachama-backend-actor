package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestAGroupReadInALanguageFallsBackToTheOriginalUntilItsTranslationIsReviewed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, err := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if _, err := mgr.SetGroupNameTranslation(ctx, g.ID, "SW", "Kata 1"); err != nil {
		t.Fatalf("SetGroupNameTranslation: %v", err)
	}
	read, err := mgr.GetGroupInLanguage(ctx, g.ID, "sw")
	if err != nil || read.Name != "Ward 1" {
		t.Fatalf("unreviewed read = %q, %v; want the original", read.Name, err)
	}
	reviewed, err := mgr.ReviewGroupNameTranslation(ctx, g.ID, "sw", "op-1")
	if err != nil || !reviewed.Reviewed || reviewed.ReviewedBy != "op-1" {
		t.Fatalf("ReviewGroupNameTranslation = %+v, %v", reviewed, err)
	}
	if read, _ := mgr.GetGroupInLanguage(ctx, g.ID, "sw"); read.Name != "Kata 1" {
		t.Errorf("reviewed read = %q, want the translation", read.Name)
	}
	if read, _ := mgr.GetGroupInLanguage(ctx, g.ID, "hu"); read.Name != "Ward 1" {
		t.Errorf("read in a language with no translation = %q, want the original", read.Name)
	}
	if read, _ := mgr.GetGroupInLanguage(ctx, g.ID, ""); read.Name != "Ward 1" {
		t.Errorf("read with no language = %q, want the original", read.Name)
	}
}

func TestChangingATranslatedNameSendsItBackForReviewAndTheSameNameDoesNot(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	_, _ = mgr.SetGroupNameTranslation(ctx, g.ID, "sw", "Kata 1")
	_, _ = mgr.ReviewGroupNameTranslation(ctx, g.ID, "sw", "op-1")
	same, err := mgr.SetGroupNameTranslation(ctx, g.ID, "sw", " Kata 1 ")
	if err != nil || !same.Reviewed {
		t.Fatalf("resending the same name = %+v, %v; want its review kept", same, err)
	}
	changed, err := mgr.SetGroupNameTranslation(ctx, g.ID, "sw", "Kata la Kwanza")
	if err != nil || changed.Reviewed || changed.ReviewedBy != "" {
		t.Fatalf("changed name = %+v, %v; want it unreviewed", changed, err)
	}
	if read, _ := mgr.GetGroupInLanguage(ctx, g.ID, "sw"); read.Name != "Ward 1" {
		t.Errorf("read after a change = %q, want the original until reviewed again", read.Name)
	}
	all, err := mgr.ListGroupNameTranslations(ctx, g.ID)
	if err != nil || len(all) != 1 || all[0].Name != "Kata la Kwanza" {
		t.Errorf("ListGroupNameTranslations = %+v, %v; want one row per language", all, err)
	}
}

func TestGroupNamesInALanguageNameOnlyReviewedTranslations(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	a, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	b, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 2"})
	_, _ = mgr.SetGroupNameTranslation(ctx, a.ID, "sw", "Kata 1")
	_, _ = mgr.SetGroupNameTranslation(ctx, b.ID, "sw", "Kata 2")
	_, _ = mgr.ReviewGroupNameTranslation(ctx, a.ID, "sw", "op-1")
	names, err := mgr.GroupNamesInLanguage(ctx, "sw", []string{a.ID, b.ID})
	if err != nil {
		t.Fatalf("GroupNamesInLanguage: %v", err)
	}
	if len(names) != 1 || names[a.ID] != "Kata 1" {
		t.Errorf("names = %v, want only the reviewed one", names)
	}
}

func TestATranslationIsRefusedWithABadLanguageAnEmptyNameAMissingGroupOrNoReviewer(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, _ := mgr.CreateGroup(ctx, mwanachamaactor.Group{Name: "Ward 1"})
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"bad language", second(mgr.SetGroupNameTranslation(ctx, g.ID, "swahili!", "Kata 1")), mwanachamaactor.ErrInvalidLanguage},
		{"empty name", second(mgr.SetGroupNameTranslation(ctx, g.ID, "sw", "  ")), mwanachamaactor.ErrInvalidTranslation},
		{"missing group", second(mgr.SetGroupNameTranslation(ctx, "nope", "sw", "Kata 1")), mwanachamaactor.ErrGroupNotFound},
		{"nothing to review", second(mgr.ReviewGroupNameTranslation(ctx, g.ID, "sw", "op-1")), mwanachamaactor.ErrTranslationNotFound},
		{"no reviewer", second(mgr.ReviewGroupNameTranslation(ctx, g.ID, "sw", "")), mwanachamaactor.ErrReviewerRequired},
	}
	for _, c := range cases {
		if !errors.Is(c.err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, c.err, c.want)
		}
	}
}

func second[T any](_ T, err error) error { return err }
