package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const redacted = "[redacted]"

var (
	uriUserinfo   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://([^/?#]*)@`)
	keywordSecret = regexp.MustCompile(`(?i)\b(?:user|password)\s*=\s*('(?:[^'\\]|\\.)*'|[^\s&]+)`)
)

func openReadOnly(dialect, location string) (*gorm.DB, func(), error) {
	config := &gorm.Config{Logger: logger.Discard}
	var dialector gorm.Dialector
	switch dialect {
	case "postgres":
		dialector = postgres.New(postgres.Config{DSN: location, PreferSimpleProtocol: true})
	case "sqlite":
		dsn, err := sqliteReadOnlyDSN(location)
		if err != nil {
			return nil, nil, err
		}
		dialector = sqlite.Open(dsn)
	default:
		return nil, nil, fmt.Errorf("the dialect is postgres or sqlite, not %q", dialect)
	}
	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to the database: %w", err)
	}
	return db, func() { closeQuietly(db) }, nil
}

func closeQuietly(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
}

func sqliteReadOnlyDSN(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("the SQLite file cannot be resolved: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("the SQLite file does not exist")
	}
	escaped := (&url.URL{Path: absolute}).EscapedPath()
	return "file:" + escaped + "?mode=ro", nil
}

func redact(message, location string) string {
	secrets := secretsIn(location)
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	for _, secret := range secrets {
		message = strings.ReplaceAll(message, secret, redacted)
	}
	return message
}

func secretsIn(location string) []string {
	secrets := []string{location}
	if abs, err := filepath.Abs(location); err == nil {
		secrets = append(secrets, abs)
	}
	if m := uriUserinfo.FindStringSubmatch(location); m != nil {
		secrets = append(secrets, m[1])
		user, password, hasPassword := strings.Cut(m[1], ":")
		secrets = append(secrets, user)
		if hasPassword {
			secrets = append(secrets, password)
			if plain, err := url.PathUnescape(password); err == nil {
				secrets = append(secrets, plain)
			}
		}
	}
	for _, m := range keywordSecret.FindAllStringSubmatch(location, -1) {
		secrets = append(secrets, m[1], strings.Trim(m[1], "'"))
	}
	kept := secrets[:0]
	for _, secret := range secrets {
		if secret != "" {
			kept = append(kept, secret)
		}
	}
	return kept
}
