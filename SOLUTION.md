# SOLUTION.md

## Decisões de Design

### Arquitetura em Fatias Verticais (Vertical Slices)

O projeto está organizado por domínio de negócio (`auth`, `teams`, `tasks`, `users`), não por camada técnica. Cada fatia contém handler, service, repository, modelos e erros — tudo que é necessário para implementar aquele caso de uso. A regra prática: se você está trabalhando em uma feature, raramente precisa sair do diretório dela.

Essa abordagem evita o anti-padrão clássico de projetos Go onde `handlers/`, `services/` e `repositories/` ficam em pacotes separados e qualquer mudança de feature exige navegar por três diretórios.

### Injeção de Dependência via Construtores

Nenhuma variável global, nenhum `init()`. Todas as dependências são injetadas explicitamente via funções construtoras (`NewService`, `NewHandler`, `NewRepository`). A composição acontece em `cmd/server/main.go`, que age como o raiz de composição da aplicação. Isso torna os componentes testáveis de forma isolada.

### Testes de Integração como Forma Principal de Teste

Em vez de mocks de banco de dados, o projeto usa `testcontainers-go` para subir um container MySQL real durante os testes. Cada teste roda dentro de uma transação GORM que é revertida via `t.Cleanup` — sem truncar tabelas, sem estado compartilhado entre testes.

Essa decisão tem custo (os testes são mais lentos), mas o benefício é que os testes exercitam queries SQL reais, constraints de banco e comportamento de GORM exatamente como em produção. Mocks de banco frequentemente deixam passar bugs que só aparecem com comportamento real do driver.

Mocks de interface são reservados apenas para o `StorageProvider` (upload de avatar), onde a alternativa seria criar arquivos reais em disco durante os testes.

### Dois Estilos de Query GORM

O projeto usa `gorm.G[T]` (API genérica type-safe) para operações simples e `db.Raw()` / `db.Table()` para queries complexas com joins e agregações. Essa escolha pragmática evita forçar todas as queries no mesmo molde: queries simples ficam legíveis, queries complexas ficam explícitas.

### StorageProvider como Interface

O upload de avatar está desacoplado do handler via interface `StorageProvider`. A implementação atual (`LocalStorage`) salva em disco. A interface permite trocar por S3, GCS ou qualquer outro provider sem mudar o handler ou o service — apenas injetar uma implementação diferente no construtor.

A detecção de tipo de arquivo usa `http.DetectContentType` nos primeiros 512 bytes (não a extensão do arquivo), o que previne que usuários renomeiem arquivos para burlar a validação.

### Tratamento de Erros com Sentinel Values

Cada fatia define seus erros de domínio como `errors.New()` (ex: `ErrTeamNotFound`, `ErrAlreadyCompleted`). Os handlers mapeiam esses erros para códigos HTTP usando `errors.Is()`. Erros de infraestrutura são sempre encapsulados com `fmt.Errorf("pacote.função.operação: %w", err)` para preservar a cadeia de erros e facilitar debugging.

### JWT Stateless

Autenticação sem sessão no servidor. O JWT contém o ID do usuário como `Subject` e expira em 24 horas. O middleware valida a assinatura e extrai o ID, que fica disponível no contexto da requisição. Sem Redis, sem tabela de sessões.

### Imagem Distroless

A imagem de produção usa `gcr.io/distroless/static-debian13:nonroot` como base. Sem shell, sem gerenciador de pacotes, sem usuário root — superfície de ataque mínima. Os binários são compilados com `CGO_ENABLED=0` (linking estático) para não depender de bibliotecas do sistema.

---

## O Que Faria Diferente com Mais Tempo


### Paginação por Cursor

A paginação atual usa offset/limit, que tem comportamento inconsistente quando registros são inseridos ou deletados entre páginas. Para listas ordenadas por score (ranking), paginação por cursor seria mais correta: `after_score` + `after_id` como parâmetros garantem resultados estáveis.

### Autorização Mais Granular

Atualmente qualquer usuário autenticado pode adicionar membros a qualquer time ou criar tarefas. Com mais tempo, implementaria roles por time (owner, member) e verificaria que apenas owners podem gerenciar membros e que apenas membros do time podem criar ou completar tarefas daquele time.


### Cache de Token Inválido

JWT stateless não tem mecanismo nativo de revogação. Com mais tempo, adicionaria uma blocklist em Redis para tokens revogados (ex: após logout ou troca de senha), com TTL igual ao tempo de expiração do token.

### sqlc em Vez de GORM

Usaria `sqlc` no lugar de um ORM. Com `sqlc`, o schema SQL é a source of truth — você escreve SQL puro, e o `sqlc` gera código Go type-safe a partir das queries. Isso elimina a camada de indireção do ORM, torna as queries explícitas e auditáveis, e remove a "mágica" de AutoMigrate. Migrations versionadas (via `golang-migrate`) complementam bem essa abordagem: o schema evolui de forma controlada, e o `sqlc` sempre reflete o estado atual. O resultado é código de acesso a dados mais previsível, mais fácil de otimizar e sem surpresas de N+1 ou comportamentos inesperados de lazy loading.

### Fronteiras de Pacotes e Interfaces no Consumer

O projeto segue a diretriz de "interfaces no ponto de uso" — cada service define a interface que precisa do seu repositório. No entanto, a estrutura atual coloca interface e implementação no mesmo pacote (`teams`), o que reduz o benefício real dessa prática.

A organização idiomática em Go seria separar os pacotes: o pacote `teams` definiria a interface `Repository` e conteria o service e o handler; um pacote separado (ex: `teamsrepo` ou `teams/mysql`) conteria a implementação concreta. Isso cria uma fronteira real: `teams` não sabe nada sobre GORM ou MySQL — depende apenas da interface. A implementação importa `teams` (para satisfazer a interface), nunca o contrário. Com `sqlc`, esse padrão fica ainda mais natural: o pacote gerado pelo `sqlc` (`db` ou `sqlcgen`) é importado apenas pelas implementações de repositório, nunca pelos services ou handlers diretamente.

---

## Trade-offs Identificados

### Testcontainers: Isolamento vs. Velocidade

**Trade-off:** Testes de integração com MySQL real são confiáveis mas lentos (30–60s para subir o container, mais o tempo de execução dos testes). Mocks seriam mais rápidos mas menos confiáveis.

**Decisão:** Priorizar confiabilidade. O container sobe uma vez por binário de teste (singleton via `sync.Once`), o que amortiza o custo. Cada teste usa uma transação isolada, então o overhead por teste é mínimo.

### Imagem Distroless vs. Operabilidade

**Trade-off:** Distroless elimina shell e ferramentas de debug. Não é possível fazer `docker exec -it container sh` para inspecionar o container em produção.

**Decisão:** Aceitar a limitação em troca de segurança. Debugging em produção deve acontecer via logs estruturados (zap) e métricas, não via shell interativo.

### AutoMigrate vs. Migrations Versionadas

**Trade-off:** GORM AutoMigrate é conveniente para desenvolvimento mas não é adequado para produção — não faz rollback, não registra histórico de migrações, e pode fazer ALTER TABLE destrutivos em schemas existentes.

**Decisão:** Usar AutoMigrate para o escopo deste desafio técnico. Em produção, substituiria por golang-migrate com arquivos SQL versionados, que permitem rollback e auditoria de schema changes.

### Vertical Slices vs. Compartilhamento de Tipos

**Trade-off:** A regra de não importar uma fatia de outra (`teams` não importa `auth`) força duplicação de tipos em alguns casos. Por exemplo, `MemberWithUser` em `teams` replica campos de `auth.User`.

**Decisão:** Aceitar a duplicação para manter o isolamento. A alternativa (um pacote `internal/types` compartilhado) cria acoplamento implícito e tende a crescer sem controle. Para este escopo, a duplicação é mínima e gerenciável.

### Credenciais Hardcoded no docker-compose vs. `.env`

**Trade-off:** O `docker-compose.yml` contém credenciais de desenvolvimento em texto plano (senhas do banco, JWT secret). Em produção, isso seria inaceitável — credenciais devem vir de um gerenciador de segredos (Vault, AWS Secrets Manager, variáveis de CI) e nunca ser commitadas.

**Decisão:** Para um desafio técnico cujo objetivo é facilitar a avaliação local, eliminar o passo de "copie o `.env.example` e preencha as variáveis" reduz a fricção. O `docker-compose.yml` é explicitamente um artefato de desenvolvimento. Em produção, esse arquivo não seria usado — o deploy aconteceria via Kubernetes, ECS ou similar, onde os segredos são injetados pelo ambiente.

### Offset Pagination vs. Cursor Pagination

**Trade-off:** Paginação por offset é simples de implementar e entender, mas tem skew quando dados mudam entre páginas. Para o ranking (ordenado por score), isso é relevante: um usuário que sobe no ranking pode aparecer em duas páginas ou não aparecer em nenhuma durante uma consulta paginada.

**Decisão:** Usar offset pagination pela simplicidade. Documentar a limitação. Em uma API com SLA de consistência, migraria para cursor pagination.
