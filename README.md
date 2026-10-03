# iot-nonna-ingest

> Part of the **iot-nonna** project. For the whole system and the Docker Compose deployment, see [iot-nonna-containers](https://github.com/Chiaf1/iot-nonna-containers).

`iot-nonna-ingest` is a backend service responsible for ingesting MQTT messages and persisting them into a PostgreSQL database in a **fully dynamic and data‑driven way**.

The service subscribes to MQTT topics, receives telemetry data from devices and sensors, decodes the payload according to metadata stored in the database, and writes the resulting values into the appropriate tables.

This repository is part of the **iot-nonna** project and represents the ingestion layer of the system.

---

## Why this service exists

In many IoT systems, MQTT topic handling and data persistence are often hard‑coded:

- topics are defined in configuration files,
- payload formats are fixed,
- adding a new sensor requires code changes and redeployment.

This approach does not scale well when:

- devices and sensors must be added or removed at runtime,
- different payload formats coexist,
- the ingestion logic must remain generic and reusable.

**iot-nonna-ingest** solves this by making the database the _single source of truth_ for:

- which MQTT topics exist,
- how payloads are interpreted,
- where data is stored,
- and how values are transformed before persistence.

---

## High‑level overview

The service works as follows:

1. **Configuration loading**
   - Service configuration (MQTT, DB, timeouts, workers) is loaded at startup.

2. **Database bootstrap**
   - A PostgreSQL connection pool is created.
   - Topic metadata is loaded from the database into an in‑memory, thread‑safe structure.

3. **MQTT connection**
   - The service connects to the MQTT broker.
   - Upon connection (and every reconnect), it subscribes to the currently active topics.

4. **Message ingestion**
   - MQTT messages are pushed into a buffered worker queue.
   - A worker pool processes messages concurrently.

5. **Dynamic decoding and persistence**
   - Each message is decoded based on metadata stored in the database:
     - payload format (`json`, `raw`, …),
     - column mapping,
     - value normalization,
     - explicit type casting.
   - Data is then inserted into the correct PostgreSQL table.

6. **Dynamic topic refresh**
   - At a configurable interval, the service re‑queries the database.
   - New topics are subscribed, removed topics are unsubscribed.
   - The topic metadata is replaced atomically without stopping ingestion.

7. **Graceful shutdown**
   - Workers, MQTT client, background goroutines and DB connections are cleanly stopped.

---

## Dynamic topic handling

### Database‑driven topic definition

Each MQTT topic is defined indirectly through database metadata.  
A `VIEW` exposes all topic information required by the ingester, including:

- MQTT topic string
- destination table
- column schema
- payload format
- qos mqtt
- optional value mapping
- device and sensor identifiers

This allows:

- adding or removing sensors without restarting the service,
- routing different topics to different tables,
- using different payload formats in parallel.

---

### Payload formats

The ingester supports multiple payload formats, defined per sensor:

- **`json`**

```json
{ "temperature": 22.5, "humidity": 61 }
```

- **`raw`**

```
        online
```

The payload format is stored in the database and interpreted dynamically at runtime.

---

### Column schema and type safety

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

- skips the column (without logging) if its key is missing from the payload,
- applies the `value_mapping` if one is defined and the value is a string,
- casts the value to the declared type (`float`, `int`, `bool`, `string`).

If a value cannot be normalized or cast, the error is logged and that column is skipped; the remaining columns, together with `device_id` and `sensor_id`, may still be inserted.

Payloads that cannot be parsed and messages on topics not present in the topic map are logged and dropped. Database insert errors are logged.

---

### Value normalization

Some payloads represent semantic values rather than technical ones.

Example (device status):

    online / offline

Through a `value_mapping` defined in the database, semantic values can be mapped dynamically:

```json
{
  "online": true,
  "offline": false
}
```

The ingester applies this mapping only when needed and remains backward‑compatible with evolving device firmware.

---

## Architecture and design principles

### Single source of truth

All ingestion behavior is driven by the database, not by code or static configuration files.

### Thread‑safe metadata access

Topic metadata is stored in a central `TopicMap`:

- fully thread‑safe,
- replaced atomically,
- safe for concurrent access by multiple workers.

### Generic worker pool

The ingestion pipeline uses a worker pool:

- MQTT callbacks are lightweight,
- heavy work is done asynchronously,
- throughput scales with the number of workers.

### Separation of concerns

- `mqtt` package: MQTT client setup and connectivity
- `postgres` package: database access
- `topic` package: topic metadata structures
- `ingestion` package: ingestion logic and orchestration
- `workers` package: concurrent message handling
- `main`: lifecycle orchestration and wiring

Each package has a single, well‑defined responsibility.

---

## MQTT subscriptions and reconnects

- Subscriptions are not static.
- On startup and on every MQTT reconnect:
  - the current topic list is re‑applied.
- Periodic refresh ensures:
  - new topics are subscribed automatically,
  - obsolete topics are unsubscribed safely.

This design is resilient to network interruptions and broker restarts.

---

## Dependency on iot-nonna-core

The database schema, the tables and the `VIEW` `mqtt_topic_list_metadata` come from the migrations of [iot-nonna-core](https://github.com/Chiaf1/iot-nonna-core). This service does not create them.

At startup the ingester queries the view to load the topic map. If this first query fails, the service logs the error and exits. For this reason `iot-nonna-core` must have run its migrations before the ingester can start.

In the [iot-nonna-containers](https://github.com/Chiaf1/iot-nonna-containers) Docker Compose setup, `ingest` only declares `depends_on` for `mqtt` and `pg_db`, not for `core`. If it starts before the migrations are done, it exits and is restarted by the restart policy until the view is available.

---

## Current status

✅ Dynamic MQTT topic handling  
✅ Database‑driven ingestion logic  
✅ Typed and validated data persistence  
✅ Concurrent worker pool  
✅ Graceful shutdown  
✅ Runtime topic refresh (polling‑based)

---

## Planned improvements

- PostgreSQL `LISTEN / NOTIFY` for real‑time topic updates
- Batch inserts for higher throughput
- Metrics and observability (Prometheus)
- Extended payload formats (binary / protobuf)
- Backpressure and retry policies

---
