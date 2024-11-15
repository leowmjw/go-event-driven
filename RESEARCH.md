# RESEARCH


## Go env

- Modulith structure
- One directory per service / bounded context

## DevX
- Taskfile - includes wait
- Profile - compatible with Heroku - Overmind - https://github.com/DarthSim/overmind

## Go Modulith

- Use multiple cmd, folders, schema per service/bounded context

## Dev Env

Compatible with Jetbrains Gateway protocol:

- Daytona - 
- Coder - 

## Deployment

- Kamal v2
- Orbstack Instance

## FrontEnd

- HTMX + FrankenUI - https://franken-ui.dev/examples/forms
- HTMX + PicoCSS - https://github.com/sonjek/go-templ-htmx-picocss-example


## Multi-tenant Operational DB with Outbox Pattern 

- MongoDB v8 w/ Atlas - local atlas cli
  - DB per tenant
  - Collection per service
  - Flat design follow single table design
  - ChangeSets - 
  - Outbox w/ Txn pattern
  - SQL Projection per query

- DynamoDB - local dynamodb 
  - Table per tenant
  - Table per service; namespaced eith tenant
  - Flat design follow single table design
  - Streaming - 
  - Outbox w/ Txn pattern
  - SQL Projection per query

- Postgres - local PG
  - Database per tenant
  - Schema per service
  - Flat design follow single txn aggregate multi-table design
  - Outbox w/ 
  - SQL Projection per query

## Orchestration

- Temporal - 
- Restate - 

## Search 

- Meilisearch - 

## Event Design

- Event Catalog v2 - 
- Event design references ..

## Event Catalog

- Boyne - 
- CloudEvent standard - 

## Lakehouse

- BemiDB - 
- Apache Pulsar - 
- Gazette -
 
