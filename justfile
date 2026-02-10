set dotenv-load

default:
    @just --list

# Start docker services
up:
    docker compose up -d
    @echo "Waiting for services..."
    @sleep 3
    @just health

# Stop docker services
down:
    docker compose down

# Restart docker services
restart: down up

# Check service health
health:
    @echo "PostgreSQL:" && docker exec ecampus-postgres pg_isready -U ecampus || echo "Not ready"
    @echo "Redis:" && docker exec ecampus-redis redis-cli ping || echo "Not ready"

# View docker logs
logs service="":
    @if [ -z "{{service}}" ]; then \
        docker compose logs -f; \
    else \
        docker compose logs -f {{service}}; \
    fi

_db_url := "postgres://" + env("DB_USER", "ecampus") + ":" + env("DB_PASSWORD", "ecampus_dev") + "@" + env("DB_HOST", "localhost") + ":" + env("DB_PORT", "5432") + "/" + env("DB_NAME", "ecampus") + "?sslmode=" + env("DB_SSLMODE", "disable")

# Run migrations
migrate-up:
    docker run --rm -v {{justfile_directory()}}/migrations:/migrations \
        --network host \
        migrate/migrate \
        -path=/migrations -database "{{_db_url}}" up

# Rollback last migration
migrate-down:
    docker run --rm -v {{justfile_directory()}}/migrations:/migrations \
        --network host \
        migrate/migrate \
        -path=/migrations -database "{{_db_url}}" down 1

# Rollback all migrations
migrate-reset:
    docker run --rm -v {{justfile_directory()}}/migrations:/migrations \
        --network host \
        migrate/migrate \
        -path=/migrations -database "{{_db_url}}" down -all

# Create new migration
migrate-create name:
    docker run --rm -v {{justfile_directory()}}/migrations:/migrations \
        migrate/migrate \
        create -ext sql -dir /migrations -seq {{name}}

# Check migration version
migrate-status:
    docker run --rm -v {{justfile_directory()}}/migrations:/migrations \
        --network host \
        migrate/migrate \
        -path=/migrations -database "{{_db_url}}" version

# Force migration version (fix dirty state)
migrate-force version:
    docker run --rm -v {{justfile_directory()}}/migrations:/migrations \
        --network host \
        migrate/migrate \
        -path=/migrations -database "{{_db_url}}" force {{version}}

# Seed database
seed:
    docker exec -i ecampus-postgres psql -U ecampus -d ecampus < scripts/seed.sql

# Reset database
db-reset: migrate-reset migrate-up seed
    @echo "Database reset complete"

# Open database CLI
db-cli:
    docker exec -it ecampus-postgres psql -U ecampus -d ecampus

# Build binary
build:
    go build -o bin/api ./cmd/api

# Build and run
run: build
    ./bin/api

# Run with hot reload
dev:
    air

# Run directly
serve:
    go run ./cmd/api

# Run all tests
test:
    go test ./... -v

# Run tests with coverage
test-coverage:
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Run tests for package
test-pkg pkg:
    go test ./internal/{{pkg}}/... -v

# Run integration tests
test-integration:
    go test ./tests/integration/... -v -tags=integration

# Run tests with race detection
test-race:
    go test ./... -race

# Format code
fmt:
    go fmt ./...

# Run linter
lint:
    golangci-lint run ./...

# Tidy modules
tidy:
    go mod tidy

# Verify dependencies
verify:
    go mod verify

# Run all quality checks
check: fmt lint verify
    @echo "All checks passed"

# Build docker image
docker-build:
    docker build -t ecampus-api:latest .

# Run docker image
docker-run:
    docker run --rm -p 8080:8080 --env-file .env --network ecampus_ecampus ecampus-api:latest

# Install dev tools
install-tools:
    go install github.com/air-verse/air@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "Tools installed"

# Clean build artifacts
clean:
    rm -rf bin/ tmp/ coverage.out coverage.html

# Setup MinIO bucket
minio-setup:
    docker exec ecampus-minio mc alias set local http://localhost:9000 minioadmin minioadmin || true
    docker exec ecampus-minio mc mb local/ecampus --ignore-existing || true
    @echo "MinIO bucket 'ecampus' created"

# Initialize .env from example
env-init:
    cp -n .env.example .env || echo ".env already exists"
