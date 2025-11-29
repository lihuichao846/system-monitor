#!/bin/bash

echo "=============================="
echo "  System Monitor Deployment"
echo "=============================="

echo "🔹 Step 1: Stop & Remove old containers..."
docker compose down

echo "🔹 Step 2: Remove dangling images..."
docker image prune -f

echo "🔹 Step 3: Rebuild images without cache..."
docker compose build --no-cache

echo "🔹 Step 4: Start containers in background..."
docker compose up -d

echo "🚀 Deployment finished! Use 'docker compose ps' to check status."
