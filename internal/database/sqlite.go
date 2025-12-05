package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"bank-aml-system/internal/config"
)

type SQLiteDB struct {
	DB *sql.DB
}

func NewSQLiteDB(cfg *config.Config) (*SQLiteDB, error) {
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

	sqlite := &SQLiteDB{DB: db}
	if err := sqlite.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("SQLite connection established")
	return sqlite, nil
}

func (s *SQLiteDB) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		processing_id TEXT UNIQUE NOT NULL,
		transaction_id TEXT NOT NULL,
		account_number TEXT NOT NULL,
		amount REAL NOT NULL,
		currency TEXT NOT NULL,
		transaction_type TEXT NOT NULL,
		counterparty_account TEXT,
		counterparty_bank TEXT,
		counterparty_country TEXT,
		timestamp DATETIME NOT NULL,
		channel TEXT,
		user_id TEXT,
		branch_id TEXT,
		status TEXT NOT NULL DEFAULT 'pending_review',
		risk_score INTEGER,
		risk_level TEXT,
		analysis_timestamp DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_processing_id ON transactions(processing_id);
	CREATE INDEX IF NOT EXISTS idx_transaction_id ON transactions(transaction_id);
	CREATE INDEX IF NOT EXISTS idx_account_number ON transactions(account_number);
	CREATE INDEX IF NOT EXISTS idx_status ON transactions(status);
	CREATE INDEX IF NOT EXISTS idx_created_at ON transactions(created_at);
	`

	_, err := s.DB.Exec(query)
	return err
}

func (s *SQLiteDB) Close() error {
	return s.DB.Close()
}

