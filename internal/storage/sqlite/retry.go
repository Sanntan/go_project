package sqlite

import (
	"fmt"
	"strings"
	"time"
)

// isRetryableError проверяет, можно ли повторить операцию при данной ошибке
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// SQLITE_BUSY (5) - база данных заблокирована
	// SQLITE_LOCKED (6) - таблица заблокирована
	return strings.Contains(errStr, "database is locked") ||
		strings.Contains(errStr, "SQLITE_BUSY") ||
		strings.Contains(errStr, "SQLITE_LOCKED") ||
		strings.Contains(errStr, "locked")
}

// retryOperation выполняет операцию с повторными попытками при ошибках блокировки
func retryOperation(operation func() error, maxRetries int, delay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Если ошибка не требует повтора, возвращаем её сразу
		if !isRetryableError(err) {
			return err
		}

		// Если это не последняя попытка, ждем перед повтором
		if i < maxRetries-1 {
			time.Sleep(delay * time.Duration(i+1)) // Экспоненциальная задержка
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}
