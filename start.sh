#!/bin/bash

# Go Order EDA Startup Script

echo "🚀 Starting Go Order EDA Application..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker Desktop first."
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose not found. Please install Docker Compose."
    exit 1
fi

echo "🔧 Building Docker images..."
docker-compose build

echo "📦 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services to be healthy..."
sleep 30

echo "🏥 Checking service health..."
docker-compose ps

echo ""
echo "✅ Go Order EDA Application is ready!"
echo ""
echo "📋 Service URLs:"
echo "  🌐 Application:        http://localhost:8080"
echo "  🏥 Health Check:       http://localhost:8080/health"
echo "  📚 API Documentation:  http://localhost:8080/api/swagger/"
echo "  🗄️  MongoDB:            mongodb://root:example@localhost:27017/order-db"
echo "  🐰 RabbitMQ Management: http://localhost:15672 (guest/guest)"
echo ""
echo "📝 Useful commands:"
echo "  📊 View logs:           docker-compose logs -f"
echo "  🛑 Stop services:       docker-compose down"
echo "  🔄 Restart:             docker-compose restart"
echo "  🧹 Clean up:            docker-compose down -v"
echo ""
echo "🎉 Happy coding!"
