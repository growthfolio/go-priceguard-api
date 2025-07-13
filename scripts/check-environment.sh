#!/bin/bash

echo "🔍 Verificando status do ambiente PriceGuard API..."
echo ""

# Verificar containers
echo "📊 Status dos containers:"
docker-compose ps
echo ""

# Verificar logs da API
echo "📋 Logs recentes da API:"
docker logs priceguard-api --tail 5
echo ""

# Testar conectividade
echo "🌐 Testando conectividade:"
echo "- Health endpoint:"
curl -s -f http://localhost:8080/health && echo "✅ API Health OK" || echo "❌ API Health FAILED"

echo "- Metrics endpoint:"
curl -s -f http://localhost:8080/metrics && echo "✅ Metrics OK" || echo "❌ Metrics FAILED"

echo ""
echo "🏁 Verificação concluída!"
