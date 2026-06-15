# Layered Architecture

This project is organized around a small layered backend structure.

## Layers

- `cmd/server`: process entrypoint and application bootstrap.
- `internal/domain/model`: domain data models shared by the application.
- `internal/application/service`: business use cases such as users, messages, groups, online state, and token blacklist.
- `internal/interfaces/http`: HTTP delivery layer, including Gin controllers, middleware, and router registration.
- `internal/interfaces/websocket`: WebSocket delivery layer and real-time message hub.
- `internal/infrastructure/config`: environment loading and external resource initialization, including MySQL, Redis, and JWT config.
- `internal/infrastructure/logger`: logging setup.

## Dependency Direction

Entrypoints wire the application together. Interface layers call application services. Application services operate on domain models and use infrastructure resources.

```text
cmd/server
  -> internal/interfaces
  -> internal/application
  -> internal/domain
  -> internal/infrastructure
```

Keep transport concerns such as Gin contexts and WebSocket connections out of application services. Keep persistence and external resource setup in infrastructure packages.

## Run

```bash
go run ./cmd/server
```
