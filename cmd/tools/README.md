# Seeder Utilities

This directory contains standalone Go commands that seed reference data into
the IAM Contracts database. Each command expects a MySQL DSN supplied via the
`--dsn` flag or the `IAM_SEEDER_DSN` environment variable and can be executed
with `go run`.

## Available commands

1. `seed-tenants` &ndash; inserts base tenant records.
2. `seed-user-center` &ndash; creates system administrators, sample users,
   children, and guardianship links.
3. `seed-auth-accounts` &ndash; configures authentication accounts along with
   operation credentials and WeChat bindings.
4. `seed-authz-resources` &ndash; registers authorization resources with their
   action sets.
5. `seed-role-assignments` &ndash; applies default role memberships to users.
6. `seed-casbin` &ndash; loads core Casbin policies and role inheritance rules.
7. `seed-jwks` &ndash; seeds JWKS key material for JWT validation.

## Example usage

```bash
export IAM_SEEDER_DSN='user:pass@tcp(127.0.0.1:3306)/iam_contracts?parseTime=true&loc=Local'
go run ./cmd/tools/seed-tenants
go run ./cmd/tools/seed-user-center
go run ./cmd/tools/seed-auth-accounts
go run ./cmd/tools/seed-authz-resources
go run ./cmd/tools/seed-role-assignments
go run ./cmd/tools/seed-casbin
go run ./cmd/tools/seed-jwks
```

Run the commands in the above order after creating an empty schema to rebuild
the baseline data set.
