package sqlite

import (
	"database/sql"

	"bank-aml-system/internal/models"
)

// GetTransactionByProcessingID получает транзакцию по processing_id
func (s *SQLiteStorage) GetTransactionByProcessingID(processingID string) (*models.TransactionStatus, error) {
	query := `
		SELECT id, processing_id, transaction_id, amount, currency, status, risk_score, 
		       risk_level, analysis_timestamp, created_at, updated_at
		FROM transactions
		WHERE processing_id = ?
	`

	var ts models.TransactionStatus
	err := s.DB.QueryRow(query, processingID).Scan(
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
func (s *SQLiteStorage) GetFullTransactionByProcessingID(processingID string) (*models.Transaction, error) {
	query := `
		SELECT transaction_id, account_number, amount, currency, transaction_type,
		       counterparty_account, counterparty_bank, counterparty_country,
		       timestamp, channel, user_id, branch_id
		FROM transactions
		WHERE processing_id = ?
	`

	var tx models.Transaction
	err := s.DB.QueryRow(query, processingID).Scan(
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
func (s *SQLiteStorage) GetAllTransactions(limit int) ([]*models.TransactionStatus, error) {
	query := `
		SELECT id, processing_id, transaction_id, amount, currency, status, risk_score, 
		       risk_level, analysis_timestamp, created_at, updated_at
		FROM transactions
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.DB.Query(query, limit)
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

