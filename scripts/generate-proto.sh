#!/bin/bash

# Скрипт для генерации Go кода из proto файлов

# Установите protoc и protoc-gen-go если их нет:
# go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

PROTO_DIR="./api/proto"
OUTPUT_DIR="./api/proto"

# Генерируем Go код из proto файлов
protoc --go_out=$OUTPUT_DIR \
       --go_opt=paths=source_relative \
       --go-grpc_out=$OUTPUT_DIR \
       --go-grpc_opt=paths=source_relative \
       $PROTO_DIR/*.proto

echo "Proto files generated successfully!"

