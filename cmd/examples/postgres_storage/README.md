# postgres_storage example

Demonstrates the Postgres session store: running migrations, creating, listing,
and deleting sessions.

## Run

```bash
# Start a local Postgres (Docker):
docker run --rm -p 5432:5432 \
  -e POSTGRES_DB=agno -e POSTGRES_USER=agno -e POSTGRES_PASSWORD=agno \
  postgres:16-alpine

# Run the example:
DATABASE_URL="postgres://agno:agno@localhost:5432/agno?sslmode=disable" \
  go run ./cmd/examples/postgres_storage/
```

If `DATABASE_URL` is unset the program logs an explanation and exits cleanly
(no error), so CI pipelines are unaffected.
