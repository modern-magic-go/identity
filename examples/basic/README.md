# Basic Example

This example wires the `identity` package to MySQL storage and optional Redis cache.

## What it does

- Creates the required MySQL tables if they do not exist.
- Seeds a demo subject, identity, and password credential by default.
- Runs password login through the public `usecase.Service`.
- Uses Redis as a cache layer when `REDIS_ADDR` is set.
- The MySQL and Redis adapters live inside this example, so they model how a consuming project would implement its own integration layer.

## Quick start

1. Start MySQL and Redis.
2. Copy `.env.example` to `.env` and adjust values if needed.
3. Run `go run ./examples/basic`.

The example will try to load `examples/basic/.env` automatically when it starts.

## Useful environment variables

- `MYSQL_DSN` - MySQL connection string. Keep `parseTime=true` enabled.
- `REDIS_ADDR` - Redis address. Leave empty to disable Redis cache.
- `REDIS_PASSWORD` - Redis password.
- `REDIS_DB` - Redis database index.
- `REDIS_TTL` - Cache TTL, for example `5m`.
- `DEMO_SEED` - Seed the demo data before login.
- `DEMO_REALM` - Demo realm.
- `DEMO_IDENTIFIER` - Demo login identifier.
- `DEMO_PASSWORD` - Demo password.
