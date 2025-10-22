#!/bin/bash
set -e

echo "🚀 Knok FM Deployment Script"
echo "================================"

# Check if .env.prod exists
if [ ! -f .env.prod ]; then
    echo "❌ Error: .env.prod file not found!"
    echo "   Copy .env.prod.example to .env.prod and fill in your values"
    exit 1
fi

# Check if running as knokfm user
if [ "$USER" != "knokfm" ] && [ "$USER" != "root" ]; then
    echo "⚠️  Warning: Not running as 'knokfm' user"
fi

echo "📦 Pulling latest code..."
git pull origin main

echo "🏗️  Building Docker images..."
docker compose -f docker-compose.prod.yml build --no-cache

echo "🛑 Stopping existing services..."
docker compose -f docker-compose.prod.yml down

echo "🚀 Starting services..."
docker compose -f docker-compose.prod.yml up -d

echo "⏳ Waiting for services to be healthy..."
sleep 10

echo "📊 Service Status:"
docker compose -f docker-compose.prod.yml ps

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📝 Next steps:"
echo "   - Check logs: docker compose -f docker-compose.prod.yml logs -f"
echo "   - Verify API: curl https://api.knok-fm.com/health"
echo "   - Visit site: https://knok-fm.com"
echo ""
