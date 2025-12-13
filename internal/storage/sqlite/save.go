package sqlite

import (
	"bank-aml-system/internal/models"
)

// SaveTransaction сохраняет транзакцию в БД со статусом pending_review
func (s *SQLiteStorage) SaveTransaction(processingID string, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (
			processing_id, transaction_id, account_number, amount, currency,
			transaction_type, counterparty_account, counterparty_bank,
			counterparty_country, timestamp, channel, user_id, branch_id, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending_review')
	`

	_, err := s.DB.Exec(
		query,
		processingID, tx.TransactionID, tx.AccountNumber, tx.Amount, tx.Currency,
		tx.TransactionType, tx.CounterpartyAccount, tx.CounterpartyBank,
		tx.CounterpartyCountry, tx.Timestamp, tx.Channel, tx.UserID, tx.BranchID,
	)

	return err
}

