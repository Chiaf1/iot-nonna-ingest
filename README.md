# MQTT Ingester
MQTT Ingester is a lightweight Go service designed to subscribe to one or more MQTT topics, buffer incoming messages, and process them asynchronously through a worker pool.

The project provides a clean separation between configuration loading, MQTT communication, and message ingestion logic.

This repository is intended to serve as a **base project** for future MQTT ingestion applications.  
The core logic is complete and production-ready, while the message processing logic included here is only a **placeholder**, meant to be replaced or extended depending on the needs of each specific project.

The intended way to use this project is to keep the core structure as-is and implement your own ingestion logic by modifying the handlers inside the ingestion module.

## Features
- Loads configuration from a YAML file, with automatic defaults and validation.
- Connects to an MQTT broker with configurable QoS and retry policies.
- Subscribes to multiple topics and pushes received messages into a buffered channel.
- Uses a worker pool to process incoming messages concurrently.
- Supports graceful shutdown:
  - stops message ingestion
  - drains and closes workers
  - disconnects from the MQTT broker
 
## Configuration
The service reads its configuration from config.yaml.
Example fields include:
- broker
- clientId
- qos
- connectionInterval
- maxRetry
- maxDelay
- topics

If the file is missing, a default one is generated automatically.

## Extending Ingestion Logic
You can add custom topic handlers inside internal/ingestion/ProcessMessage.
Each topic can route to a different processing function, database writer, or transformation pipeline.

```
switch msg.Topic {
case "sensors/temperature":
    handleTemperature(msg)
case "metrics/power":
    handlePower(msg)
}
```
