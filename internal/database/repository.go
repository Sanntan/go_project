package database

import (
	"database/sql"
	"time"

	"bank-aml-system/internal/models"
)

type Repository struct {
	db *SQLiteDB
}

func NewRepository(db *SQLiteDB) *Repository {
	return &Repository{db: db}
}

// SaveTransaction сохраняет транзакцию в БД со статусом pending_review
func (r *Repository) SaveTransaction(processingID string, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (
			processing_id, transaction_id, account_number, amount, currency,
			transaction_type, counterparty_account, counterparty_bank,
			counterparty_country, timestamp, channel, user_id, branch_id, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending_review')
	`

	_, err := r.db.DB.Exec(
		query,
		processingID, tx.TransactionID, tx.AccountNumber, tx.Amount, tx.Currency,
		tx.TransactionType, tx.CounterpartyAccount, tx.CounterpartyBank,
		tx.CounterpartyCountry, tx.Timestamp, tx.Channel, tx.UserID, tx.BranchID,
	)

	return err
}

// UpdateTransactionAnalysis обновляет результаты анализа транзакции
func (r *Repository) UpdateTransactionAnalysis(
	processingID string,
	riskScore int,
	riskLevel string,
	analysisTime time.Time,
) error {
	query := `
		UPDATE transactions
		SET status = 'reviewed',
		    risk_score = ?,
		    risk_level = ?,
		    analysis_timestamp = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE processing_id = ?
	`

	_, err := r.db.DB.Exec(query, riskScore, riskLevel, analysisTime, processingID)
	return err
}

// GetTransactionByProcessingID получает транзакцию по processing_id
func (r *Repository) GetTransactionByProcessingID(processingID string) (*models.TransactionStatus, error) {
	query := `
		SELECT id, processing_id, transaction_id, amount, currency, status, risk_score, 
		       risk_level, analysis_timestamp, created_at, updated_at
		FROM transactions
		WHERE processing_id = ?
	`

	var ts models.TransactionStatus
	err := r.db.DB.QueryRow(query, processingID).Scan(
		&ts.ID, &ts.ProcessingID, &ts.TransactionID, &ts.Amount, &ts.Currency, &ts.Status,
		&ts.RiskScore, &ts.RiskLevel, &ts.AnalysisTimestamp,
		&ts.CreatedAt, &ts.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &ts, nil
}

// GetFullTransactionByProcessingID получает полную транзакцию со всеми полями
func (r *Repository) GetFullTransactionByProcessingID(processingID string) (*models.Transaction, error) {
	query := `
		SELECT transaction_id, account_number, amount, currency, transaction_type,
		       counterparty_account, counterparty_bank, counterparty_country,
		       timestamp, channel, user_id, branch_id
		FROM transactions
		WHERE processing_id = ?
	`

	var tx models.Transaction
	err := r.db.DB.QueryRow(query, processingID).Scan(
		&tx.TransactionID, &tx.AccountNumber, &tx.Amount, &tx.Currency, &tx.TransactionType,
		&tx.CounterpartyAccount, &tx.CounterpartyBank, &tx.CounterpartyCountry,
		&tx.Timestamp, &tx.Channel, &tx.UserID, &tx.BranchID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

// GetAllTransactions получает все транзакции из БД
func (r *Repository) GetAllTransactions(limit int) ([]*models.TransactionStatus, error) {
	query := `
		SELECT id, processing_id, transaction_id, amount, currency, status, risk_score, 
		       risk_level, analysis_timestamp, created_at, updated_at
		FROM transactions
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*models.TransactionStatus
	for rows.Next() {
		var ts models.TransactionStatus
		err := rows.Scan(
			&ts.ID, &ts.ProcessingID, &ts.TransactionID, &ts.Amount, &ts.Currency, &ts.Status,
			&ts.RiskScore, &ts.RiskLevel, &ts.AnalysisTimestamp,
			&ts.CreatedAt, &ts.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, &ts)
	}

	return transactions, rows.Err()
}

// ClearAllTransactions удаляет все транзакции из БД
func (r *Repository) ClearAllTransactions() error {
	query := `DELETE FROM transactions`
	_, err := r.db.DB.Exec(query)
	return err
}

