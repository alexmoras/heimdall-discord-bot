#!/bin/bash

# Heimdall Setup Script
# This script helps you set up the Heimdall Discord bot

set -e

echo "🛡️  Heimdall Discord Bot Setup"
echo "================================"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.23 or higher first."
    echo "   Visit: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✅ Go $GO_VERSION detected"
echo ""

# Check if config.yaml exists
if [ ! -f "config.yaml" ]; then
    echo "📝 Creating config.yaml from example..."
    cp config.yaml.example config.yaml
    echo "✅ config.yaml created"
    echo ""
    echo "⚠️  IMPORTANT: Please edit config.yaml with your settings before running the bot!"
    echo "   You need to configure:"
    echo "   - Discord bot token"
    echo "   - Guild (server) ID"
    echo "   - SMTP email settings"
    echo "   - Base URL for your server"
    echo "   - Approved email domains"
    echo "   - Team roles"
    echo ""
    read -p "Press Enter to continue after you've edited config.yaml..."
else
    echo "✅ config.yaml already exists"
fi

echo ""
echo "📦 Installing dependencies..."
go mod download
echo "✅ Dependencies installed"
echo ""

echo "🔨 Building Heimdall..."
go build -o heimdall
echo "✅ Build complete"
echo ""

echo "🎉 Setup complete!"
echo ""
echo "To run Heimdall:"
echo "  ./heimdall"
echo ""
echo "To run as a service (Linux):"
echo "  1. sudo cp heimdall.service /etc/systemd/system/"
echo "  2. sudo systemctl daemon-reload"
echo "  3. sudo systemctl enable heimdall"
echo "  4. sudo systemctl start heimdall"
echo ""
echo "To run with Docker:"
echo "  docker-compose up -d"
echo ""
echo "For more information:"
echo "  - Quick start guide: QUICKSTART.md"
echo "  - Full documentation: README.md"
echo "  - Moderator commands: MODERATOR_COMMANDS.md"
