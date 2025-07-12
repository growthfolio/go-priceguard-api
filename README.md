# 🛡️ PriceGuard API

Sistema avançado de monitoramento de criptomoedas em tempo real com alertas inteligentes e análise técnica.

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![API Status](https://img.shields.io/badge/API-Active%20Development-orange.svg)]()

## 🚀 Visão Geral

O PriceGuard é uma API REST robusta construída em Go que oferece:

- **💡 Alertas Inteligentes**: Sistema avançado com múltiplas condições (preço, RSI, percentual, cruzamento de médias)
- **⚡ Real-time**: WebSocket com broadcasting automático de alertas e notificações
- **📊 Análise Técnica**: Indicadores completos (RSI, EMA, SuperTrend, MACD, Bollinger)
- **🔔 Notificações**: Sistema assíncrono com fila Redis e retry automático
- **🔐 Autenticação**: Google OAuth 2.0 + JWT com refresh tokens
- **🏗️ Clean Architecture**: Estrutura modular e escalável

## 📈 Status do Projeto

**Fase Atual**: 8/15 ✅ **Sistema de Alertas Implementado**

| Fase | Status | Descrição |
|------|--------|-----------|
| 1-3 | ✅ | Estrutura inicial e modelagem |
| 4 | ✅ | Autenticação JWT + Google OAuth |
| 5 | ✅ | APIs REST Core |
| 6 | ✅ | Integração Binance + Indicadores |
| 7 | ✅ | WebSocket Real-time |
| **8** | ✅ | **Sistema de Alertas Avançado** |
| 9 | 🚧 | Middleware e Segurança |
| 10-15 | ⏳ | Monitoramento, Testes, Deploy |

> 📋 [Ver progresso detalhado](./DEVELOPMENT_CHECKLIST.md)

## 🛠️ Stack Tecnológica

| Categoria | Tecnologia |
|-----------|------------|
| **Backend** | Go 1.21+, Gin Framework |
| **Database** | PostgreSQL, Redis |
| **Real-time** | WebSocket (Gorilla) |
| **APIs** | Binance API, Google OAuth |
| **DevOps** | Docker, Air (hot reload) |

## ⚡ Quick Start

```bash
# 1. Clone o repositório
git clone https://github.com/growthfolio/go-priceguard-api.git
cd go-priceguard-api

# 2. Configure ambiente
cp .env.example .env
# Edite .env com suas configurações

# 3. Execute com Docker
make docker-up

# Ou execute localmente
make run
```

## 🏗️ Arquitetura

```
📁 cmd/server/          # Entry point
📁 internal/
  📁 adapters/          # HTTP, WebSocket, Repository
  📁 application/       # Services (AlertEngine, NotificationService)
  📁 domain/           # Entities (User, Alert, Notification)
  📁 infrastructure/   # Database, External APIs
📁 pkg/                # Utilities (Indicators)
```

## 🔗 API Endpoints

<details>
<summary><strong>📡 Principais Endpoints</strong></summary>

### Autenticação
- `POST /auth/login` - Login Google OAuth
- `POST /auth/refresh` - Refresh token

### Alertas
- `GET /api/alerts` - Listar alertas
- `POST /api/alerts` - Criar alerta
- `GET /api/alerts/types` - Tipos disponíveis
- `GET /api/alerts/stats` - Estatísticas

### Notificações  
- `GET /api/notifications` - Listar notificações
- `POST /api/notifications/mark-all-read` - Marcar como lidas
- `GET /api/notifications/stats` - Estatísticas

### Dados Crypto
- `GET /api/crypto/data` - Lista de moedas
- `GET /api/crypto/history/:symbol` - Histórico de preços
- `GET /api/crypto/indicators/:symbol` - Indicadores técnicos

### WebSocket
- `WS /ws` - Conexão real-time
  - Events: `alert_triggered`, `notification_update`, `crypto_data_update`

</details>

## 🔧 Configuração

<details>
<summary><strong>⚙️ Variáveis de Ambiente</strong></summary>

```env
# Servidor
PORT=8080
GIN_MODE=debug

# Database
DB_HOST=localhost
DB_NAME=priceguard
DB_USER=postgres
DB_PASSWORD=your_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=your_super_secret_key
JWT_EXPIRATION=24h

# Google OAuth
GOOGLE_CLIENT_ID=your_client_id
GOOGLE_CLIENT_SECRET=your_client_secret

# Binance API
BINANCE_API_KEY=your_api_key
BINANCE_API_SECRET=your_api_secret
```

</details>

## 🧪 Comandos

```bash
make run          # Executar aplicação
make build        # Build da aplicação  
make test         # Executar testes
make docker-up    # Docker compose up
make docker-down  # Docker compose down
make migrate-up   # Executar migrações
```

## 📚 Documentação

- 📖 [Especificação Técnica](./docs/BACKEND_SPECIFICATION.md)
- ✅ [Checklist de Desenvolvimento](./DEVELOPMENT_CHECKLIST.md)
- 🔗 [Swagger API Docs](http://localhost:8080/swagger/index.html)

## 🤝 Contribuindo

1. Fork o projeto
2. Crie uma branch (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -m 'feat: adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

## 👨‍💻 Desenvolvimento

**Desenvolvido por**: Felipe Macedo / GitHub Copilot  
**Repositório**: [github.com/growthfolio/go-priceguard-api](https://github.com/growthfolio/go-priceguard-api)  
**Licença**: MIT

---

⭐ **Se este projeto foi útil, considere dar uma estrela!**
