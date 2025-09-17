# Modbus ETH Controller

A compact Modbus TCP controller for use with
[Waveshare’s Modbus POE ETH Relay](https://www.waveshare.com/wiki/Modbus_POE_ETH_Relay) 
[non-affiliate Amazon link](https://a.co/d/9HrijBM) and similar Ethernet-connected relay devices.

Supports:

- Direct Modbus TCP commands to read, write, and toggle relay coils
- Declarative, JSON-based "programs" for complex patterns
- HTTP API for integration with home automation platforms like Home Assistant
- CLI interface for one-off commands and other use cases
- Multi-architecture from-scratch (small!) Docker image (amd64 + arm64)
- Devices with any number of relay coils (1-65536)
- Stored and ad-hoc programs

Not supported:

- Other Modbus functions (only coil read/write)
- Modbus RTU (serial)
- Authentication or encryption (e.g. TLS)

---

## 🧰 Use Cases

- Ring a mechanical doorbell from a smart doorbell trigger
- Toggle relays to control garage doors, lights, etc.
- Execute coordinated patterns across multiple relays
- Integrate with automations and webhook events

---

## 🚀 Running via Docker

### One-off execution (reads program from stdin or file)

```bash
docker run --rm -v $(pwd)/programs:/etc/modbus jakerobb/modbus-eth-controller:latest < my-program.json
```

### Using Docker Compose

You can use Docker Compose to run the controller with a mounted configuration directory:

```yaml
version: '3'
services:
  modbus-eth-controller:
    image: jakerobb/modbus-eth-controller:latest
    volumes:
      - ./programs:/etc/modbus
    ports:
      - "8080:8080"
```

Replace `./programs` with the path to your directory containing JSON program files. You can omit this volume, in which 
case the application will start with four built-in example programs.

Run with:

```bash
docker-compose up -d
```

---

## 🌐 HTTP API

The controller exposes a REST-ish HTTP API for integration:

- `GET /status` - Returns the current status of the relays.
- `GET /programs` - Lists available programs in the mounted directory.
- `POST /run` - Accepts a JSON program to execute immediately.
- `POST /run?program=name` - Executes one or more saved programs by name. (Provide the `program` query parameter multiple times to run multiple programs in sequence.)

The `/status` endpoint first performs a binary search to determine the number of available relays. This takes at most
sixteen modbus messages, and is cached for subsequent calls. Then we return the status of each relay (`true`=on, `false`=off).
Every call to `/run` includes a `status` field in the response containing this same data.

The `/run` endpoint accepts a single program in the request body, and any number of named programs in the query parameters.
The program in the request body is executed first, and the rest in the order they are provided in the query string.

Programs in the mounted directory are loaded upon startup and cached in memory. Changes to the files will be picked up 
in a just-in-time fashion; programs are reloaded from disk if the file modification date is newer than what was loaded.

When loaded, each file is assigned a "slug". This is a normalized version of the filename. Slugs are always lowercase, 
and anything that isn't a letter or a digit will be replaced with a hyphen. For example, `My Doorbell.json` becomes
`my-doorbell`. Files not ending with `.json` are ignored. Subdirectories are not scanned.

Actual slugs are logged during startup, or you can call the /programs endpoint to see them. 

By default, the HTTP port is 8080, and the program listens on all available interfaces. You can override these using 
environment variables (see below), or you can of course map them to whatever you like using Docker.

No attempt is made to secure the HTTP API. If you expose it to untrusted networks, I recommend wrapping it in a reverse 
proxy with encryption and some form of authentication.

---

## 📄 Program Format

Programs are JSON files describing relay patterns and sequences. Example:

```json
{
  "address": "modbus.lan:4196",
  "commandIntervalMillis": 200,
  "loops": 2,
  "commands": [
    [ { "command": "on", "relay": 7 } ],
    [ { "command": "off", "relay": 7 } ]
  ]
}
```

`address` specifies the Modbus device (IP or hostname and port). My modbus relay defaulted to port 4196; I make no 
promises about yours. I use my Unifi gateway to create a DNS entry for the device, hence `modbus.lan`. You can use an IP
address or any name you like, so long as it resolves from within the Docker container.

`commandIntervalMillis` sets the delay between command groups.

`commands` is an array of command groups. Each group is an array of commands to execute in parallel.

Commands can be:
- `on` - Turn a relay on
- `off` - Turn a relay off
- `toggle` - Toggle a relay's state (note: this is not standard Modbus protocol, but my Waveshare device supports it)

Relay numbers are one-indexed (e.g. 1-8).

This program turns one relay on, waits 200ms, then turns it off. It's the main reason I built this; I plan to use it to
ring a mechanical doorbell from a Unifi G6 Entry, which does not have a standard doorbell output like the older G4 
Doorbell.

Here's a more complex program that runs all eight relays in sequence, leaving three on at a time, in a 
Christmas-light-style "chasing"  sequence. It loops twenty times. Adjust `commandIntervalMillis` and `loops` to control 
the speed and total duration.

```json
{
  "address": "modbus.lan:4196",
  "commandIntervalMillis": 80,
  "loops": 20,
  "commands": [
    [ { "command": "on", "relay": 0 }, { "command": "off", "relay": 6 } ],
    [ { "command": "on", "relay": 1 }, { "command": "off", "relay": 7 } ],
    [ { "command": "on", "relay": 2 }, { "command": "off", "relay": 0 } ],
    [ { "command": "on", "relay": 3 }, { "command": "off", "relay": 1 } ],
    [ { "command": "on", "relay": 4 }, { "command": "off", "relay": 2 } ],
    [ { "command": "on", "relay": 5 }, { "command": "off", "relay": 3 } ],
    [ { "command": "on", "relay": 6 }, { "command": "off", "relay": 4 } ],
    [ { "command": "on", "relay": 7 }, { "command": "off", "relay": 5 } ]
  ]
}
```

## Responses

The /run endpoint responds with the program(s) executed, along with some information about the execution. 

```json
{
  "results": [
    {
      "status": "success",
      "startTime": "2024-09-13T12:34:56.123456Z",
      "executionTimeMillis": 205,
      "slug": "doorbell",
      "program": { ... }
    }
  ],
  "status": {
    "modbus.lan:4196": {
      "coils": {
        "1": false,
        "2": false,
        "3": false,
        "4": false,
        "5": false,
        "6": false,
        "7": false,
        "8": false
      }
    }
  }
]
```

`slug` is the program slug, or `"(ad-hoc)"` for programs sent in the request body.

If there were errors, the `status` field will be `"error"`, and an `error` field will contain details.

---

## ⚙️ Environment Variables

- `MODBUS_PROGRAM_DIR` - Directory for JSON programs (default: `/etc/modbus`)
- `LISTEN_PORT` - Port for HTTP API (default: `8080`)
- `LISTEN_ADDRESS` - Interface address on which the program will listen (default: `0.0.0.0`, i.e. all interfaces).

These variables are only relevant when running with the `--server` option.

---

## 🐳 Docker Image and Tags

The docker image is published to Docker Hub at [jakerobb/modbus-eth-controller](https://hub.docker.com/r/jakerobb/modbus-eth-controller).

- `latest` - Latest stable release
- older releases are tagged with the date they were released, e.g. 20250913

Published images are multi-architecture (amd64 + arm64).

---

## 🛠 Development

Clone the repository and build the Docker image locally:

```bash
git clone https://github.com/jakerobb/modbus-eth-controller.git
cd modbus-eth-controller
docker build -t modbus-eth-controller .
```

---

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
