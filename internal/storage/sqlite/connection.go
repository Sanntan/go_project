package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"bank-aml-system/config"
)

// SQLiteStorage представляет хранилище SQLite
type SQLiteStorage struct {
	DB *sql.DB
}

// NewConnection создает новое соединение с SQLite
func NewConnection(cfg *config.Config) (*SQLiteStorage, error) {
	// Определяем путь к файлу БД
	dbPath := cfg.DB.DBPath
	if dbPath == "" {
		// Используем путь по умолчанию в текущей директории
		dbPath = "./data/bank_aml.db"
	}

	// Создаем директорию, если её нет
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Формируем DSN для SQLite
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=1", dbPath)

	log.Printf("Connecting to SQLite: path=%s", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(1) // SQLite поддерживает только одно соединение для записи
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	storage := &SQLiteStorage{DB: db}
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("SQLite connection established")
	return storage, nil
}

// Close закрывает соединение с БД
func (s *SQLiteStorage) Close() error {
	return s.DB.Close()
}

