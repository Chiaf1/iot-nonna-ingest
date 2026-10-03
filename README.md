# iot-nonna-ingest

> Part of the **iot-nonna** project. For the whole system and the Docker Compose deployment, see [iot-nonna-containers](https://github.com/Chiaf1/iot-nonna-containers).

`iot-nonna-ingest` is a backend service that receives MQTT messages and inserts readings into PostgreSQL using topic metadata stored in the database.

The metadata defines the topics, payload formats, column mappings and destination tables. The code implements the supported formats and type conversions; destination tables must already exist.

This repository is part of the **iot-nonna** project and is the ingestion layer of the system.

---

## Why this service exists

In many IoT systems, topic handling and data persistence are hard-coded:

- topics are defined in configuration files,
- payload formats are fixed,
- adding a new sensor requires code changes and redeployment.

This service moves topic-specific settings into the database:

- which MQTT topics to subscribe to,
- how supported payloads are interpreted,
- which existing table and columns receive the readings,
- which string values are mapped before insertion.

Devices and sensor associations can be added or removed through metadata changes without restarting the ingester. New payload formats or conversion types still require code changes, and new destination tables require schema changes.

---

## High-level overview

The service works as follows:

1. **Configuration loading**
   - Service configuration (MQTT, DB, timeouts, workers) is loaded at startup.
   - `CONFIG_PATH` selects the configuration file; the default is `./config.yaml`.

2. **Database bootstrap**
   - A PostgreSQL connection pool is created.
   - Topic metadata is loaded from the database into an in-memory `TopicMap` protected by an `RWMutex`.

3. **MQTT connection**
   - The service connects to the MQTT broker.
   - On connection and reconnect, it attempts to subscribe to the topics currently in the map.

4. **Message ingestion**
   - MQTT callbacks send messages into a buffered queue with capacity for 1,000 messages.
   - A configurable number of workers process messages concurrently.

5. **Dynamic decoding and persistence**
   - Each message is processed using its topic metadata:
     - payload format (`json` or `raw`),
     - column mapping,
     - optional string-value mapping,
     - conversion to a supported type.
   - The handler attempts an insert into the configured PostgreSQL table.

6. **Dynamic topic refresh**
   - At the configured interval, while MQTT is connected, the service queries the database again.
   - It subscribes to new topics and requests unsubscription from removed topics.
   - If loading or subscribing fails, the refresh returns an error without replacing the map.
   - Otherwise, it replaces the metadata map under a lock. The unsubscribe result is not awaited or checked.

7. **Shutdown handling**
   - On `SIGINT` or `SIGTERM`, the service cancels the refresher context, closes the worker queue, requests MQTT unsubscription, disconnects the client and closes the DB pool.
   - It does not explicitly wait for the workers or the refresher goroutine to finish. This does not guarantee that queued messages are persisted before exit.

---

## Dynamic topic handling

### Database-driven topic definition

Each MQTT topic is defined through database metadata. The `mqtt_topic_list_metadata` view exposes:

- MQTT topic string,
- destination table,
- column schema,
- payload format,
- optional MQTT QoS,
- optional value mapping,
- device and sensor identifiers.

Within the supported formats and types, this allows:

- adding or removing sensor associations without restarting the service,
- routing different topics to different existing tables,
- using `json` and `raw` payloads in parallel.

---

### Payload formats

The ingester supports two payload formats, defined per sensor:

- **`json`**: a JSON object whose keys are matched against `column_schema`.

```json
{ "temperature": 22.5, "humidity": 61 }
```

- **`raw`**: the complete payload is treated as a string under the key `$payload`.

```text
online
```

For a raw payload, `column_schema` must use `$payload` as its source key. Other format names return a parsing error.

---

### Column schema and type conversion

For each topic, the database defines a `column_schema` describing:

- the payload key,
- the target database column,
- the expected data type.

Example:

```json
{
  "temperature": { "column": "temperature", "type": "float" },
  "humidity": { "column": "humidity", "type": "float" }
}
```

For each column defined in `column_schema`, the ingester:

- skips the column without logging if its key is missing from the payload,
- applies `value_mapping` if one is defined and the value is a string,
- converts or checks the value according to the declared type (`float`, `int`, `bool`, `string`).

The current conversions are:

- `float`: accepts a Go `float64`, including numbers decoded from JSON.
- `int`: accepts a `float64` and converts it to `int64`; fractional values are truncated, not rejected.
- `bool`: accepts a Go `bool`.
- `string`: formats the value with `fmt.Sprintf("%v", val)`.

If mapping or conversion fails, the error is logged and that column is skipped. The handler still attempts to insert the remaining columns together with `device_id` and `sensor_id`. If every payload column is skipped, it still attempts an insert containing only those two identifiers; database constraints determine whether the insert succeeds.

Payload parsing errors and messages on topics absent from the map are logged and dropped. Database insert errors are logged; the handler does not retry the insert.

These checks are not full reading validation. They do not enforce sensor ranges or require all configured payload keys to be present.

---

### Value normalization

Some payloads use strings to represent values.

Example (device status):

```text
online / offline
```

A `value_mapping` can convert them before type conversion:

```json
{
  "online": true,
  "offline": false
}
```

If no mapping is defined, the value is passed through. Non-string values are also passed through unchanged. When a mapping is defined and the value is a string, the ingester lowercases the string and looks it up in the mapping. It does not trim whitespace. A missing mapping key causes that column to be logged and skipped.

---

## Architecture and design principles

### Metadata-driven routing

Topics, destination tables, column mappings, payload-format selection and value mappings come from the database. Supported parsers and type conversions are implemented in code; connections, worker count and refresh timing come from the service configuration.

### Locked metadata access

Topic metadata is stored in a central `TopicMap`:

- reads use an `RWMutex` read lock,
- getters copy the column-schema and value-mapping maps,
- `ReplaceMap` prepares a copy, then swaps the map and updates its timestamp under the write lock.

This protects the map replacement from concurrent map reads. It does not make MQTT subscription changes and metadata replacement one atomic operation.

### Generic worker pool

The ingestion pipeline uses a worker pool:

- MQTT callbacks enqueue the topic and payload,
- workers perform parsing, conversion and database inserts,
- the worker count is configurable.

Sending to the queue blocks when its buffer is full. The service has no explicit worker-completion wait during shutdown.

### Separation of concerns

- `mqtt` package: MQTT client setup and connectivity
- `postgres` package: database access
- `topic` package: topic metadata structures
- `ingestion` package: ingestion logic and topic refresh
- `workers` package: queue and concurrent message handling
- `main`: startup, shutdown and wiring

---

## MQTT subscriptions and reconnects

- On connection and reconnect, the service attempts to subscribe to every topic currently in the map.
- Subscription QoS comes from topic metadata when available, otherwise from the configured fallback. Values above 2 also use the fallback.
- Periodic refresh subscribes to newly added topics and requests unsubscription from removed topics.
- Unsubscribe tokens are not awaited or checked, so removal from the local map does not confirm that broker-side unsubscription succeeded.
- Metadata changes for existing topics are loaded on refresh, but changed QoS is not re-applied to an existing subscription until the next reconnect.

The MQTT client enables automatic reconnect. Subscription and refresh errors are logged; this is not a guarantee of uninterrupted delivery or successful persistence.

---

## Dependency on iot-nonna-core

The database schema, tables and `mqtt_topic_list_metadata` view come from the migrations of [iot-nonna-core](https://github.com/Chiaf1/iot-nonna-core). This service does not create them.

At startup, the ingester queries the view to load the topic map. If this first query fails, the service logs the error and exits. The migrations must therefore have been applied before the ingester can start successfully. The core API does not need to remain running for ingestion once the required schema exists.

In the [iot-nonna-containers](https://github.com/Chiaf1/iot-nonna-containers) Docker Compose setup, `ingest` declares `depends_on` only for `mqtt` and `pg_db`, not for `core`. It can start before the migrations are done and exit. The Compose file sets `restart: unless-stopped`; check the logs and restart `ingest` after the migrations if needed.

---

## Current status

- Database-driven MQTT topic selection and routing
- `json` and `raw` payload parsing
- Per-column value mapping and type conversion, with failed columns skipped
- Concurrent worker pool with a buffered queue
- Signal-triggered shutdown sequence, without an explicit worker-completion wait
- Polling-based metadata and subscription refresh

---

## Planned improvements

- PostgreSQL `LISTEN / NOTIFY` for event-driven topic updates
- Batch inserts
- Metrics and observability (Prometheus)
- Additional payload formats (binary / protobuf)
- Explicit backpressure and insert retry policies

---
