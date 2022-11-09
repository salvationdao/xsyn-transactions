# XSYN Transaction System

This repo has two binaries:

- Migrator
- Server

The migrator's purpose is to connect to the legacy transaction Postgres DB and move all the data to a new database.
The server will host a REST API that can be used to register accounts and transfers.

- Source database: XSYN Postgres
- Target database: XSYN TimescaleDB

It uses protobufs to generate the handlers.

Dependency-wise, it SHOULD be completely separate from the XSYN server.

Spin-up
If you are working on this codebase `make init` will initialize the codebase including spinning up the database with migrations and data migration.
If you are not working on the codebase you want to just run `make docker-serve`

Envars
```sh
## Migrator
XSYN_TRANSACTIONS_MIGRATE_FROM_DB_USER=
XSYN_TRANSACTIONS_MIGRATE_FROM_DB_PASS=
XSYN_TRANSACTIONS_MIGRATE_FROM_DB_HOST=
XSYN_TRANSACTIONS_MIGRATE_FROM_DB_PORT=
XSYN_TRANSACTIONS_MIGRATE_FROM_DB_NAME=
XSYN_TRANSACTIONS_MIGRATE_TO_DB_USER=
XSYN_TRANSACTIONS_MIGRATE_TO_DB_PASS=
XSYN_TRANSACTIONS_MIGRATE_TO_DB_HOST=
XSYN_TRANSACTIONS_MIGRATE_TO_DB_PORT=
XSYN_TRANSACTIONS_MIGRATE_TO_DB_NAME=


## API
XSYN_TRANSACTIONS_DB_USER=
XSYN_TRANSACTIONS_DB_PASS=
XSYN_TRANSACTIONS_DB_HOST=
XSYN_TRANSACTIONS_DB_PORT=
XSYN_TRANSACTIONS_DB_NAME=
XSYN_TRANSACTIONS_DB_MAX_IDLE_CONNS=
XSYN_TRANSACTIONS_DB_MAX_OPEN_CONNS=
XSYN_TRANSACTIONS_API_PORT=
XSYN_TRANSACTIONS_AUTH_KEY=# this is the key clients need to provide to connect to the service


## buf related
BUF_TOKEN="1pass"
```

## Generate Code

- `buf generate`
    - generates protobuf code locally
- `buf push`
  - pushes proto files to registry 
- https://buf.build/


Make Commands
```bash
# sets up go tools and sets up docker db
make init

# db commands
make db-reset
make db-drop
make db-migrate
make db-migrate-down
make db-migrate-down-one
make db-migrate-up-one
make db-boiler

# this command migrates transaction data from xsyn-services, also gets ran on db-reset
make migrate-from-old

# buf commands
make buf

# container commands
# this runs the "dev" profile from the compose file. It runs db/migrations/data-migrations
make docker-serve-dev
make docker-stop-dev
# runs all containers in compose file including api service
make docker-serve
# tears down and removes the containers
make docker-down
# builds DockerfileXsynTransactions and DockerfileMigrate
make docker-build
```
