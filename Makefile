.PHONY: all run build test clean init-db run-scheduler deps full-setup help docker-build docker-up docker-down docker-logs docker-ps docker-clean docker-init-db docker-add-user docker-shell docker-restart docker-up-dev

# All commands
all: build

# Default goal: show help
.DEFAULT_GOAL := help

# Show help
help:
	@echo ""
	@echo "========================================"
	@echo "  Daily Email Sender - Makefile Commands"
	@echo "========================================"
	@echo ""
	@echo "  make init-db      - Initialize database"
	@echo "  make run-scheduler- Start the scheduler"
	@echo "  make add-user     - Add user interactively"
	@echo "  make list-users   - Show list of users"
	@echo "  make add-schedule - Add schedule interactively"
	@echo "  make full-setup   - Full user creation cycle"
	@echo "  make build        - Build executable file"
	@echo "  make run          - Show this help"
	@echo "  make deps         - Download dependencies"
	@echo "  make clean        - Clean up (remove binary)"
	@echo "  make test         - Test compilation"
	@echo ""
	@echo "----------------------------------------"
	@echo "  Go CLI commands (via ./daily-email-sender.exe)"
	@echo "----------------------------------------"
	@echo ""
	@echo "  add-user          - Add a new user"
	@echo "  list-users        - Show all users"
	@echo "  add-schedule      - Add schedule for a user"
	@echo "  run-scheduler     - Start the scheduler"
	@echo "  init-db           - Initialize database"
	@echo "  help              - Show this help"
	@echo ""
	@echo "----------------------------------------"
	@echo "  Examples"
	@echo "----------------------------------------"
	@echo ""
	@echo "  make build && make add-user"
	@echo "  make run-scheduler"
	@echo "  make list-users"
	@echo "  make init-db && make add-user && make run-scheduler"
	@echo ""

# Build executable file
build:
	@echo "Building..."
	go build -o daily-email-sender.exe .
	@echo "Build complete: daily-email-sender.exe"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "Dependencies downloaded"

# Clean up
clean:
	@echo "Cleaning..."
	rm -f daily-email-sender.exe
	@echo "Clean complete"

# Test compilation
test:
	@echo "Testing compilation..."
	go build -o /dev/null .
	@echo "Test passed"

# Initialize database
init-db:
	@echo "Initializing database..."
	go run . init-db
	@echo "Database initialized"

# Start scheduler
run-scheduler:
	@echo "Starting scheduler..."
	go run . run-scheduler

# Add user interactively
add-user:
	@echo "Adding user..."
	go run . add-user

# Show list of users
list-users:
	@echo "Listing users..."
	go run . list-users

# Add schedule interactively
add-schedule:
	@echo "Adding schedule..."
	go run . add-schedule

# Full user creation cycle
full-setup:
	@echo "Running full setup..."
	$(MAKE) add-user

# Docker targets
docker-build:
	@echo "Building Docker images..."
	docker-compose build
	@echo "Build complete!"

docker-up:
	@echo "Starting Docker containers..."
	docker-compose up -d
	@echo "Containers started! PostgreSQL is available at localhost:5432"

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down
	@echo "Containers stopped!"

docker-logs:
	docker-compose logs -f

docker-ps:
	docker-compose ps

docker-clean:
	@echo "Cleaning Docker containers and volumes..."
	docker-compose down -v
	@echo "Clean complete!"

docker-init-db:
	@echo "Initializing database inside containers..."
	docker-compose exec app ./daily-email-sender init-db
	@echo "Database initialized!"

docker-add-user:
	@echo "Adding user..."
	docker-compose exec app ./daily-email-sender add-user
	@echo "User added!"

docker-shell:
	docker-compose exec app sh

docker-restart:
	@echo "Restarting containers..."
	docker-compose restart
	@echo "Restart complete!"

# Docker development setup
docker-up-dev:
	@echo "Starting Docker containers in development mode..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo "Containers started in development mode!"
	@echo "To initialize and add user, run: make docker-dev-setup"
