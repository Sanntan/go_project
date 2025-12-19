package main

import "bank-aml-system/internal/bootstrap/ingestion"

// @title Bank AML System API
// @version 1.0
// @description Система противодействия отмыванию денег и мошенничеству
// @host localhost:8080
// @BasePath /api/v1
func main() { ingestion.StartIngestionService() }
