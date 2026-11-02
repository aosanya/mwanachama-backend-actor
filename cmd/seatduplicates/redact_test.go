package main

import (
	"strings"
	"testing"
)

func TestRedactStrikesTheLocationItsPasswordItsUserAndItsHost(t *testing.T) {
	location := "postgres://admin:p%40ss@db.example.com:5432/postgres"

	got := redact("failed: "+location+" and admin:p%40ss and p@ss and user=admin reaching db.example.com:5432 via db.example.com", location)

	for _, leak := range []string{"p%40ss", "p@ss", "admin", "db.example.com", location} {
		if strings.Contains(got, leak) {
			t.Errorf("redacted message still holds %q: %s", leak, got)
		}
	}
	if !strings.Contains(got, "failed:") || !strings.Contains(got, "reaching") {
		t.Errorf("redacted message lost the words around the secrets: %s", got)
	}
}

func TestRedactStrikesTheHostOfAKeywordAddress(t *testing.T) {
	location := "host=db.internal port=5432 user=bob password=hunter22 dbname=hub"

	got := redact("dial tcp db.internal:5432: i/o timeout for bob with hunter22", location)

	for _, leak := range []string{"db.internal", "bob", "hunter22"} {
		if strings.Contains(got, leak) {
			t.Errorf("redacted message still holds %q: %s", leak, got)
		}
	}
}

func TestRedactStrikesTheHostOfAnAddressThatDoesNotParse(t *testing.T) {
	got := redact("cannot parse port in 127.0.0.1:notaport", "postgres://u:pw@127.0.0.1:notaport/postgres")

	if strings.Contains(got, "127.0.0.1") {
		t.Errorf("redacted message still holds the host: %s", got)
	}
}

func TestRedactLeavesWordsThatOnlyContainAShortSecretAlone(t *testing.T) {
	got := redact("a banana and a cat cannot connect", "postgres://a:b@h/db")

	if want := "[redacted] banana and [redacted] cat cannot connect"; got != want {
		t.Errorf("redact = %q, want %q", got, want)
	}
}

func TestRedactDoesNotStrikeInsideItsOwnMarker(t *testing.T) {
	location := "host=127.0.0.1 port=1 user=a password=x dbname=db sslmode=disable"

	got := redact("a cannot reach 127.0.0.1 with x", location)

	if want := "[redacted] cannot reach [redacted] with [redacted]"; got != want {
		t.Errorf("redact = %q, want %q", got, want)
	}
}

func TestAShortUserLeavesAReadableErrorAndNoHost(t *testing.T) {
	for name, location := range map[string]string{
		"uri":     "postgres://a:b@127.0.0.1:1/db",
		"keyword": "host=127.0.0.1 port=1 user=a password=x dbname=db sslmode=disable",
	} {
		got := runWith(map[string]string{"POSTGRES_URL": location}, "-instance", testInstance)
		if got.code != 2 {
			t.Errorf("%s: exit %d, want 2: %+v", name, got.code, got)
		}
		if !strings.Contains(got.stderr, "cannot connect to the database") {
			t.Errorf("%s: the error is no longer readable: %q", name, got.stderr)
		}
		if strings.Contains(got.stderr, "127.0.0.1") {
			t.Errorf("%s: the error names the host: %q", name, got.stderr)
		}
	}
}
