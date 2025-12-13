package sqlite

import (
	"time"
)

// UpdateTransactionAnalysis обновляет результаты анализа транзакции
func (s *SQLiteStorage) UpdateTransactionAnalysis(
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

	_, err := s.DB.Exec(query, riskScore, riskLevel, analysisTime, processingID)
	return err
}

