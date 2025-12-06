package database

import (
	"os"
	"testing"
	"time"

	"bank-aml-system/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteDB(t *testing.T) {
	tmpFile := "test_new_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: tmpFile,
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Проверяем, что БД создана
	_, err = os.Stat(tmpFile)
	assert.NoError(t, err)
}

func TestNewSQLiteDB_DefaultPath(t *testing.T) {
	// Создаем временную директорию
	tmpDir := "test_data_" + time.Now().Format("20060102150405")
	defer os.RemoveAll(tmpDir)

	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: "", // Пустой путь - должен использоваться дефолтный
		},
	}

	// Устанавливаем текущую директорию для теста
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.MkdirAll(tmpDir, 0755)
	os.Chdir(tmpDir)

	// Создаем БД с дефолтным путем
	dbPath := "./data/bank_aml.db"
	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Проверяем, что БД создана
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Очищаем
	os.RemoveAll("./data")
}

func TestNewSQLiteDB_InMemory(t *testing.T) {
	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: ":memory:",
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Проверяем, что можем выполнить запрос
	var result int
	err = db.DB.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)
}

func TestSQLiteDB_InitSchema(t *testing.T) {
	tmpFile := "test_schema_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: tmpFile,
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Проверяем, что таблица transactions создана
	var tableName string
	err = db.DB.QueryRow(`
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name='transactions'
	`).Scan(&tableName)
	require.NoError(t, err)
	assert.Equal(t, "transactions", tableName)

	// Проверяем, что индексы созданы
	var indexCount int
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='index' AND tbl_name='transactions'
	`).Scan(&indexCount)
	require.NoError(t, err)
	assert.Greater(t, indexCount, 0)
}

func TestSQLiteDB_Close(t *testing.T) {
	tmpFile := "test_close_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: tmpFile,
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)

	// Закрываем БД
	err = db.Close()
	require.NoError(t, err)

	// Проверяем, что после закрытия нельзя выполнить запрос
	err = db.DB.Ping()
	assert.Error(t, err)
}

func TestSQLiteDB_ConnectionPool(t *testing.T) {
	tmpFile := "test_pool_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: tmpFile,
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Проверяем настройки пула соединений
	stats := db.DB.Stats()
	assert.Equal(t, 1, stats.MaxOpenConnections)
}

func TestSQLiteDB_TableStructure(t *testing.T) {
	tmpFile := "test_structure_" + time.Now().Format("20060102150405") + ".db"
	defer os.Remove(tmpFile)

	cfg := &config.Config{
		DB: config.DBConfig{
			DBPath: tmpFile,
		},
	}

	db, err := NewSQLiteDB(cfg)
	require.NoError(t, err)
	defer db.Close()

	// Проверяем структуру таблицы transactions
	rows, err := db.DB.Query("PRAGMA table_info(transactions)")
	require.NoError(t, err)
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		require.NoError(t, err)

		columns[name] = true
	}

	// Проверяем наличие основных колонок
	requiredColumns := []string{
		"id", "processing_id", "transaction_id", "account_number",
		"amount", "currency", "transaction_type", "status",
		"created_at", "updated_at",
	}

	for _, col := range requiredColumns {
		assert.True(t, columns[col], "Column %s should exist", col)
	}
}

