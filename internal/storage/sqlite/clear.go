package sqlite

// ClearAllTransactions удаляет все транзакции из БД
func (s *SQLiteStorage) ClearAllTransactions() error {
	query := `DELETE FROM transactions`
	_, err := s.DB.Exec(query)
	return err
}

