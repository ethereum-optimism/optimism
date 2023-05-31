## @eth-optimism/indexer-ui

A simple UI for exploring the indexer DB using [Prisma studio](https://www.prisma.io)

## Usage

Included in the docker-compose file as `ui` service

```bash
docker-compose up
```

Prisma can be viewed at [localhost:5555](http://localhost:5555)

## Update the schema

The [prisma schema](https://www.prisma.io/docs/reference/api-reference/prisma-schema-reference) is what allows prisma to work. It is automatically generated from the db schema.

To update the schema to the latest db schema simply pass in the database url to [prisma pull](https://www.prisma.io/docs/reference/api-reference/command-reference#db-pull). Prisma pull will introspect the schema and generate a prisma schema

```bash
DATABASE_URL=postgresql://db_username:db_password@postgres:5432/db_name npx prisma db pull
```

## Other functionality

We mostly just use prisma as a UI. But brisma provides much other functionality that can be useful including.

- Ability to change the [db schema](https://www.prisma.io/docs/reference/api-reference/command-reference#db-push) direction from modifying the [schema.prisma](./schema.prisma) in place. This can be a fast way to [start prototyping](https://www.prisma.io/docs/guides/migrate/prototyping-schema-db-push)
- Ability to [seed the database](https://www.prisma.io/docs/guides/migrate/seed-database)
- Ability to write quick scripts with [prisma client](https://www.prisma.io/docs/reference/api-reference/prisma-client-reference)

## Running prisma studio outside of docker

Prisma can also be run with [npx](https://docs.npmjs.com/cli/v8/commands/npx)

```bash
npx prisma studio --schema indexer/ui/schema.prisma
```
