# Database Migrations

This directory holds `sql-migrate` migration files.

## Usage

Install `sql-migrate`:

```bash
go install github.com/rubenv/sql-migrate/...@latest
```

Run migrations:

```bash
export DATABASE_URL=postgres://devuser:devpass@localhost:5432/wacrm?sslmode=disable
sql-migrate up
```

Create a new migration:

```bash
sql-migrate new -env=development <name>
```

Configuration is in `dbconfig.yml` at the project root.
