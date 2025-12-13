package sqlite

// initSchema инициализирует схему БД
func (s *SQLiteStorage) initSchema() error {
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

