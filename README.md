# Taskify

API REST para gerenciamento de tarefas em times. Permite criar times, adicionar membros, criar e completar tarefas, e consultar o ranking de pontuação dos membros.

## Stack

| Tecnologia | Versão |
|---|---|
| Go | 1.26 |
| Echo | v5 |
| GORM | v2 |
| MySQL | 8.x |
| JWT | golang-jwt/jwt v5 |
| Logger | go.uber.org/zap |

---

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) e [Docker Compose](https://docs.docker.com/compose/install/)
- Go 1.26+ (apenas para rodar testes ou o servidor localmente sem Docker)

---

## Rodando com Docker Compose

As variáveis de ambiente já estão configuradas no `docker-compose.yml` para desenvolvimento local. Nenhuma configuração adicional é necessária.

### Suba os containers

```bash
docker compose up --build
```

Isso sobe três serviços em ordem:

1. **mysql** — banco MySQL 8.4 com healthcheck
2. **migrate** — roda as migrations (GORM AutoMigrate) e encerra
3. **api** — servidor HTTP na porta configurada em `PORT`

### 3. Acesse a API

```
http://localhost:8080
```

Documentação interativa com todos os endpoints disponível em:

```
http://localhost:8080/docs
```

## Rodando os testes

Os testes de integração sobem um container MySQL real via [testcontainers-go](https://golang.testcontainers.org/). É necessário ter o Docker rodando.

```bash
go test -race -count=1 ./...
```

> `-count=1` desativa o cache do Go para garantir que os containers sempre rodem frescos.

---

## Estrutura do projeto

```
.
├── api/
│   ├── embed.go          # Embeds openapi.yaml no binário
│   └── openapi.yaml      # Spec OpenAPI 3.1
├── cmd/
│   ├── migrate/          # Binário de migration (GORM AutoMigrate)
│   └── server/           # Binário principal do servidor
├── internal/
│   ├── auth/             # Registro e login de usuários
│   ├── docs/             # Handler de documentação (Redocly + spec YAML)
│   ├── infra/            # Banco de dados, validação, erros HTTP
│   ├── middleware/        # JWT middleware
│   ├── tasks/            # CRUD de tarefas
│   ├── teams/            # Times, membros e ranking
│   ├── testhelper/       # Helpers de teste (container, transações, factories)
│   ├── users/            # Avatar de usuário e StorageProvider
│   └── validate/         # Validação de structs
├── docker-compose.yml
├── Dockerfile
└── SOLUTION.md           # Decisões de design e trade-offs
```
