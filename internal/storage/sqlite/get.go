package sqlite

import (
	"database/sql"
	"time"

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
// С retry логикой для обработки race condition (транзакция может еще не сохраниться)
func (s *SQLiteStorage) GetFullTransactionByProcessingID(processingID string) (*models.Transaction, error) {
	query := `
		SELECT transaction_id, account_number, amount, currency, transaction_type,
		       counterparty_account, counterparty_bank, counterparty_country,
		       timestamp, channel, user_id, branch_id
		FROM transactions
		WHERE processing_id = ?
	`

	var tx *models.Transaction
	var lastErr error

	// Retry для случая, когда транзакция еще не сохранилась (race condition)
	for i := 0; i < 5; i++ {
		var result models.Transaction
		err := s.DB.QueryRow(query, processingID).Scan(
			&result.TransactionID, &result.AccountNumber, &result.Amount, &result.Currency, &result.TransactionType,
			&result.CounterpartyAccount, &result.CounterpartyBank, &result.CounterpartyCountry,
			&result.Timestamp, &result.Channel, &result.UserID, &result.BranchID,
		)

		if err == sql.ErrNoRows {
			// Транзакция еще не найдена - возможно race condition
			if i < 4 { // Не последняя попытка
				time.Sleep(time.Duration(i+1) * 50 * time.Millisecond) // Экспоненциальная задержка
				continue
			}
			return nil, nil // После всех попыток возвращаем nil
		}

		if err != nil {
			// Если ошибка блокировки, повторяем
			if isRetryableError(err) && i < 4 {
				lastErr = err
				time.Sleep(time.Duration(i+1) * 50 * time.Millisecond)
				continue
			}
			return nil, err
		}

		// Успешно получили транзакцию
		tx = &result
		break
	}

	if tx == nil && lastErr != nil {
		return nil, lastErr
	}

	return tx, nil
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

