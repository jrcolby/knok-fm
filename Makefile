# Knock FM Development Makefile

.PHONY: help dev-services dev-bot dev-api build clean test

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev-services: ## Start development PostgreSQL and Redis
	docker-compose -f docker-compose.dev.yml up -d
	@echo "✅ Development services started"
	@echo "   PostgreSQL: localhost:5432 (user: dev, pass: devpass, db: knockfm)"
	@echo "   Redis: localhost:6379"
	@echo "   Redis UI: http://localhost:8081"

dev-services-stop: ## Stop development services
	docker-compose -f docker-compose.dev.yml down

dev-bot: ## Run Discord bot in development mode
	@echo "🤖 Starting Discord bot..."
	@echo "📋 Loading environment..."
	@if [ -f .env ]; then echo "✅ Found .env file"; else echo "⚠️  No .env file found"; fi
	@source .env 2>/dev/null || true && \
	echo "🔑 Environment variables:" && \
	echo "  DATABASE_URL: postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" && \
	echo "  REDIS_URL: redis://localhost:6379" && \
	echo "  LOG_LEVEL: debug" && \
	echo "  DISCORD_TOKEN: $${DISCORD_TOKEN:0:20}..." && \
	echo "🚀 Starting bot service..." && \
	DATABASE_URL="postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	LOG_LEVEL="debug" \
	DISCORD_TOKEN="$$DISCORD_TOKEN" \
	go run cmd/bot/main.go

dev-worker: ## Run background worker in development mode
	@echo "⚙️  Starting background worker..."
	@echo "📋 Loading environment..."
	@if [ -f .env ]; then echo "✅ Found .env file"; else echo "⚠️  No .env file found"; fi
	@source .env 2>/dev/null || true && \
	echo "🔑 Environment variables:" && \
	echo "  DATABASE_URL: postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" && \
	echo "  REDIS_URL: redis://localhost:6379" && \
	echo "  LOG_LEVEL: debug" && \
	echo "  DISCORD_TOKEN: $${DISCORD_TOKEN:0:20}... (worker doesn't need this)" && \
	echo "🚀 Starting worker service..." && \
	DATABASE_URL="postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	LOG_LEVEL="debug" \
	go run cmd/worker/main.go

dev-api: ## Run API server in development mode
	@echo "🌐 Starting API server..."
	@echo "📋 Loading environment..."
	@if [ -f .env ]; then echo "✅ Found .env file"; else echo "⚠️  No .env file found"; fi
	@source .env 2>/dev/null || true && \
	echo "🔑 Environment variables:" && \
	echo "  DATABASE_URL: postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" && \
	echo "  REDIS_URL: redis://localhost:6379" && \
	echo "  LOG_LEVEL: debug" && \
	echo "  PORT: 8080" && \
	echo "  ADMIN_API_KEY: $${ADMIN_API_KEY:+[SET]}$${ADMIN_API_KEY:-[NOT SET]}" && \
	echo "🚀 Starting API service..." && \
	DATABASE_URL="postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	LOG_LEVEL="debug" \
	PORT="8080" \
	ADMIN_API_KEY="$$ADMIN_API_KEY" \
	go run cmd/api/main.go

seed-channel: ## Seed database from Discord channel (requires -channel and -guild flags)
	@echo "🌱 Seeding from Discord channel..."
	@echo "📋 Loading environment..."
	@if [ -f .env ]; then echo "✅ Found .env file"; else echo "⚠️  No .env file found"; fi
	@source .env 2>/dev/null || true && \
	echo "🔑 Environment variables:" && \
	echo "  DATABASE_URL: postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" && \
	echo "  REDIS_URL: redis://localhost:6379" && \
	echo "  LOG_LEVEL: info" && \
	echo "  DISCORD_TOKEN: $${DISCORD_TOKEN:0:20}..." && \
	echo "" && \
	echo "Usage: make seed-channel ARGS='-channel CHANNEL_ID -guild GUILD_ID [OPTIONS]'" && \
	echo "Options:" && \
	echo "  -channel ID    Discord channel ID (required)" && \
	echo "  -guild ID      Discord server/guild ID (required)" && \
	echo "  -limit N       Max messages to fetch (default: no limit)" && \
	echo "  -batch N       Messages per API call (default: 100, max: 100)" && \
	echo "  -before ID     Fetch messages before this message ID" && \
	echo "  -after ID      Fetch messages after this message ID" && \
	echo "  -dry-run       Preview without creating knoks" && \
	echo "" && \
	if [ -z "$(ARGS)" ]; then \
		echo "❌ Error: ARGS not provided"; \
		echo "Example: make seed-channel ARGS='-channel 123456789 -guild 987654321'"; \
		exit 1; \
	fi && \
	echo "🚀 Starting seeder with args: $(ARGS)" && \
	DATABASE_URL="postgresql://dev:devpass@localhost:5432/knockfm?sslmode=disable" \
	REDIS_URL="redis://localhost:6379" \
	LOG_LEVEL="info" \
	DISCORD_TOKEN="$$DISCORD_TOKEN" \
	go run cmd/seeder/main.go $(ARGS)

build: ## Build all binaries
	@echo "🔨 Building binaries..."
	@mkdir -p bin
	go build -o bin/bot ./cmd/bot
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	go build -o bin/seeder ./cmd/seeder
	@echo "✅ Built: bin/bot, bin/api, bin/worker, bin/seeder"

test: ## Run tests
	go test ./...

lint: ## Run linter
	go vet ./...
	go fmt ./...

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

setup-env: ## Create example .env file
	@if [ ! -f .env ]; then \
		echo "Creating .env file..."; \
		echo "PORT=8080" > .env; \
		echo "LOG_LEVEL=debug" >> .env; \
		echo "STATIC_DIR=./web/build" >> .env; \
		echo "DATABASE_URL=postgresql://dev:devpass@localhost:5432/knockfm" >> .env; \
		echo "REDIS_URL=redis://localhost:6379" >> .env; \
		echo "DISCORD_TOKEN=your_discord_bot_token_here" >> .env; \
		echo "✅ Created .env file - please edit DISCORD_TOKEN"; \
	else \
		echo "⚠️  .env file already exists"; \
	fi

# Development workflow
dev: dev-services setup-env ## Full development setup
	@echo "🚀 Development environment ready!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Edit .env and set your DISCORD_TOKEN"
	@echo "2. Run 'make dev-api' to start API server"
	@echo "3. Run 'make dev-bot' to start Discord bot"

# Production deployment commands (configure via environment variables)
# Create a .env.prod.local file (gitignored) with your settings:
#   PROD_SSH_HOST - SSH host (e.g., user@yourserver.com or ssh alias)
#   PROD_APP_DIR - Path to app on server (default: ~/knok-fm)
#   DISCORD_GUILD_ID - Your Discord server ID for seeding
#   DISCORD_CHANNEL_ID - Your Discord channel ID for seeding

# Load .env.prod.local if it exists (gitignored)
-include .env.prod.local
export

PROD_SSH_HOST ?= $(shell echo $$PROD_SSH_HOST)
PROD_APP_DIR ?= ~/knok-fm
DISCORD_GUILD_ID ?= $(shell echo $$DISCORD_GUILD_ID)
DISCORD_CHANNEL_ID ?= $(shell echo $$DISCORD_CHANNEL_ID)

check-prod-config:
	@if [ -z "$(PROD_SSH_HOST)" ]; then \
		echo "❌ Error: PROD_SSH_HOST not set"; \
		echo "Set it in your environment: export PROD_SSH_HOST=user@yourserver.com"; \
		exit 1; \
	fi

deploy: check-prod-config ## Deploy to production (rebuild and restart all services)
	@echo "🚀 Deploying to production..."
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && git pull origin main && docker compose -f docker-compose.prod.yml build && docker compose -f docker-compose.prod.yml up -d'
	@echo "✅ Deployment complete!"

deploy-worker: check-prod-config ## Deploy only worker service
	@echo "⚙️  Deploying worker..."
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && git pull origin main && docker compose -f docker-compose.prod.yml build worker && docker compose -f docker-compose.prod.yml up -d worker'
	@echo "✅ Worker deployed!"

deploy-web: check-prod-config ## Deploy only web service
	@echo "🌐 Deploying web..."
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && git pull origin main && docker compose -f docker-compose.prod.yml build web && docker compose -f docker-compose.prod.yml up -d web'
	@echo "✅ Web deployed!"

logs: check-prod-config ## Tail production logs (all services)
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml logs -f'

logs-worker: check-prod-config ## Tail worker logs
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml logs -f worker'

logs-bot: check-prod-config ## Tail bot logs
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml logs -f bot'

logs-api: check-prod-config ## Tail API logs
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml logs -f api'

prod-restart: check-prod-config ## Restart all production services
	@echo "🔄 Restarting services..."
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml restart'
	@echo "✅ Services restarted!"

prod-status: check-prod-config ## Show production service status
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml ps'

prod-nuke: check-prod-config ## Nuke database and reseed (DESTRUCTIVE!)
	@if [ -z "$(DISCORD_GUILD_ID)" ] || [ -z "$(DISCORD_CHANNEL_ID)" ]; then \
		echo "❌ Error: DISCORD_GUILD_ID and DISCORD_CHANNEL_ID must be set for seeding"; \
		echo "Set them in your environment or .env file"; \
		exit 1; \
	fi
	@echo "⚠️  WARNING: This will delete all data!"
	@read -p "Are you sure? Type 'yes' to continue: " confirm && [ "$$confirm" = "yes" ] || exit 1
	@echo "💥 Nuking database..."
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml down && docker volume rm knok-fm_postgres_data && docker compose -f docker-compose.prod.yml up -d'
	@echo "⏳ Waiting for services to start..."
	@sleep 10
	@echo "🌱 Seeding database..."
	ssh $(PROD_SSH_HOST) 'cd $(PROD_APP_DIR) && docker compose -f docker-compose.prod.yml exec api ./seeder -channel $(DISCORD_CHANNEL_ID) -guild $(DISCORD_GUILD_ID) -limit 0'
	@echo "✅ Database nuked and reseeded!"
