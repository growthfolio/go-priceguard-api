# PriceGuard API

> **Projeto pessoal de estudo, mantido como registro histórico.**
> Não é um serviço financeiro em produção, não representa experiência profissional
> com operação de sistemas em escala e não deve ser interpretado como benchmark de
> capacidade, segurança ou disponibilidade.

Este repositório foi criado para praticar desenvolvimento backend em Go e explorar,
em um ambiente de laboratório, temas como APIs HTTP, WebSocket, PostgreSQL, Redis,
autenticação, containers e organização de código.

## O que este projeto representa

O objetivo principal foi aprender construindo. O repositório reúne código,
configurações e experimentos relacionados a:

- APIs REST em Go;
- comunicação WebSocket;
- persistência com PostgreSQL;
- uso de Redis em cenários de cache/fila;
- autenticação com OAuth/JWT;
- Docker e arquivos de estudo para Kubernetes;
- testes e documentação de API;
- organização de código inspirada em Clean Architecture.

Termos como **microserviços**, **sistemas distribuídos**, **Kubernetes** e
**observabilidade** aparecem porque foram assuntos estudados durante o projeto.
Isso não significa experiência profissional em produção com toda essa stack.

## Estado

O projeto é um laboratório pessoal e não recebe garantia de compatibilidade,
segurança ou operação contínua. Algumas partes podem exigir serviços externos,
credenciais ou ajustes locais para execução.

Afirmações antigas de "production-ready", "enterprise-grade", milhares de conexões,
uptime e throughput foram removidas deste README porque não devem ser apresentadas
sem um processo atual, reproduzível e independente de validação.

## Execução local

O repositório possui arquivos de desenvolvimento com Docker e Make. Para explorar
o projeto, revise primeiro as variáveis de ambiente de exemplo e os serviços
externos exigidos pelo código antes de executar qualquer comando.

```bash
git clone https://github.com/growthfolio/go-priceguard-api.git
cd go-priceguard-api
cp .env.example .env
make -f Makefile.dev dev
```

Os comandos disponíveis podem mudar conforme o estado do repositório:

```bash
make -f Makefile.dev help
```

## Contexto

Este projeto faz parte de uma fase de estudos e experimentação. Meu perfil atual,
experiência profissional e projetos que continuo desenvolvendo estão concentrados
em:

- https://felipemacedo.me
- https://github.com/felipemacedo1

## Licença

Consulte o arquivo [LICENSE](LICENSE).
