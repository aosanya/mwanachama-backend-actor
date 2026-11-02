package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

func withJournalHint(dialect, location string, err error) error {
	if dialect != "sqlite" || !strings.Contains(err.Error(), "readonly database") || !inWALMode(location) {
		return err
	}
	return fmt.Errorf("%w; the file is in WAL mode, so reading it needs a directory where its -wal and -shm files can be made", err)
}

func inWALMode(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	header := make([]byte, 20)
	if _, err := io.ReadFull(file, header); err != nil {
		return false
	}
	return header[18] == 2 && header[19] == 2
}
