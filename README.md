# Delcom Blog — Backend (Golang Gin)

REST API blog multi-tier + AI generator (Search Engine + OpenAI). Port default **8081**.

## Menjalankan
```bash
cp .env.example .env          # sesuaikan DB & OPENAI_KEY / OPENAI_MODEL
psql -U postgres -c "CREATE DATABASE pabwe_p9_blog;"
go mod tidy                   # sekali saja, lalu commit go.sum
go run ./cmd/server
```
Swagger UI: http://localhost:8081/swagger/index.html

## Test (TDD, SQLite in-memory — tidak butuh PostgreSQL)
```bash
go test -v -cover ./internal/... ./docs/...
```

## Build
```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o server ./cmd/server
```
Akun seeder: `admin/admin` (admin), `member/member` (member).
