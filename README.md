# PriceGuard API Backend

Backend em Go para a aplicação PriceGuard - Monitor de preços de criptomoedas em tempo real.

## 📋 Sobre o Projeto

O PriceGuard é uma aplicação de monitoramento de criptomoedas que fornece:

- ⚡ Dados em tempo real via WebSocket
- 📊 Indicadores técnicos (RSI, EMA, SuperTrend, etc.)
- 🔔 Sistema de alertas personalizáveis
- 🔐 Autenticação via Google OAuth
- 📈 Histórico de preços e análises
- 🎛️ Dashboard interativo

## 🚀 Status do Desenvolvimento

Este projeto está em desenvolvimento ativo. Consulte o [DEVELOPMENT_CHECKLIST.md](./DEVELOPMENT_CHECKLIST.md) para acompanhar o progresso.

**Fase Atual**: Estrutura Inicial e Configuração

## 🏗️ Arquitetura

O backend segue os princípios de Clean Architecture com as seguintes camadas:

```
📁 cmd/server/          # Ponto de entrada
📁 internal/
  📁 adapters/          # Adaptadores (HTTP, WebSocket, Repository)
  📁 application/       # Casos de uso e serviços de aplicação
  📁 domain/           # Entidades e regras de negócio
  📁 infrastructure/   # Infraestrutura externa (DB, APIs)
📁 pkg/                # Pacotes utilitários
📁 docs/               # Documentação
```

## 🛠️ Tecnologias

- **Linguagem**: Go 1.21+
- **Framework HTTP**: Gin
- **Database**: PostgreSQL
- **Cache**: Redis
- **WebSocket**: Gorilla WebSocket
- **Autenticação**: JWT + Google OAuth 2.0
- **APIs Externas**: Binance API
- **Containerização**: Docker

## 📋 Funcionalidades Principais

### APIs REST
- ✅ Autenticação e autorização
- ✅ Gestão de perfil de usuário
- ✅ Dados de criptomoedas
- ✅ Sistema de alertas
- ✅ Notificações

### WebSocket Real-time
- ✅ Atualizações de preços em tempo real
- ✅ Indicadores técnicos ao vivo
- ✅ Alertas instantâneos
- ✅ Sistema de subscrições

### Indicadores Técnicos
- RSI (Relative Strength Index)
- EMA (Exponential Moving Average)
- SuperTrend
- True Range
- Pullback Entry Signals

## 🚀 Desenvolvimento Local

### Pré-requisitos

- Go 1.21+
- PostgreSQL 14+
- Redis 6+
- Docker (opcional)

### Configuração

1. Clone o repositório:
```bash
git clone <repo-url>
cd go-priceguard-api
```

2. Configure as variáveis de ambiente:
```bash
cp .env.example .env
# Edite o arquivo .env com suas configurações
```

3. Execute as migrações do banco:
```bash
make migrate-up
```

4. Execute a aplicação:
```bash
make run
```

### Comandos Disponíveis

```bash
make run          # Executar aplicação
make build        # Build da aplicação
make test         # Executar testes
make test-cover   # Testes com coverage
make lint         # Linter
make docker-up    # Subir containers
make docker-down  # Parar containers
```

## 📡 Endpoints da API

### Autenticação
- `POST /auth/login` - Login via Google OAuth
- `POST /auth/logout` - Logout
- `GET /auth/verify` - Verificar token

### Usuário
- `GET /api/user/profile` - Perfil do usuário
- `PUT /api/user/profile` - Atualizar perfil
- `GET /api/user/settings` - Configurações
- `PUT /api/user/settings` - Atualizar configurações

### Criptomoedas
- `GET /api/crypto/data` - Lista de criptomoedas
- `GET /api/crypto/detail/:symbol` - Detalhes específicos
- `GET /api/crypto/history/:symbol` - Histórico
- `GET /api/crypto/indicators/:symbol` - Indicadores

### Alertas
- `GET /api/alerts` - Listar alertas
- `POST /api/alerts` - Criar alerta
- `PUT /api/alerts/:id` - Atualizar alerta
- `DELETE /api/alerts/:id` - Excluir alerta

### WebSocket
- `WS /ws/dashboard` - Conexão para dados em tempo real

## 🔧 Variáveis de Ambiente

```env
# Servidor
PORT=8080
GIN_MODE=debug

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=priceguard

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your_jwt_secret
JWT_EXPIRATION=24h

# Google OAuth
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret

# Binance API
BINANCE_API_KEY=your_binance_api_key
BINANCE_API_SECRET=your_binance_api_secret
```

## 🧪 Testes

```bash
# Executar todos os testes
make test

# Testes com coverage
make test-cover

# Testes específicos
go test ./internal/...
```

## 📚 Documentação

- [Especificação do Backend](./docs/BACKEND_SPECIFICATION.md)
- [Checklist de Desenvolvimento](./DEVELOPMENT_CHECKLIST.md)
- [Documentação da API](./docs/api/) (Swagger)

## 🤝 Contribuição

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

## 📄 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## 📞 Contato

Para dúvidas ou sugestões, entre em contato com a equipe de desenvolvimento.

---

⭐ **Desenvolvido com Go e ❤️**
