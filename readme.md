# XSYN Transaction System

This sub-repo has two binaries:

- Migrator
- Server

The migrator's purpose is to connect to the legacy transaction Postgres DB and move all the data to a new database.
The server will host a REST API that can be used to register accounts and transfers.

- Source database: XSYN Postgres
- Target database: XSYN Timescale or Tigerbeetle

It uses protobufs to generate the handlers.

Dependency-wise, it SHOULD be completely separate from the XSYN server.

## Generate Code

- `buf generate`
- https://buf.build/
