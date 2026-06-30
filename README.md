
# KickCraft

A web-based tool for generating automated installation ISOs for Rocky Linux and CentOS using Kickstart configuration files.

## Features

- Web UI for Kickstart configuration
- Support for Rocky Linux 8/9/10 and CentOS 7/8
- Visual partition editor
- Network configuration
- Package selection
- Pre/post installation scripts
- Template system
- ISO generation

## Supported Distributions

### Rocky Linux
| Version | Status | EOL Date |
|---------|--------|----------|
| Rocky 8 | Active | 2029-05 |
| Rocky 9 | Active | 2032-05 |
| Rocky 10 | Beta | TBD |

### CentOS
| Version | Status | EOL Date |
|---------|--------|----------|
| CentOS 7 | EOL | 2024-06 |
| CentOS 8 | EOL | 2021-12 |

## Quick Start

### Run with Docker (Recommended)

```bash
docker-compose up -d
```

Then access the web UI at http://localhost:8080

### Run Locally

```bash
# Install dependencies
make deps

# Run
make run
```

### Build from Source

```bash
# Build
make build

# Run the binary
./kickcraft --port 8080
```

## Docker images (Recommended)

**Suggestion:**

- The version of your Docker image must match the version of the image you want to build; otherwise, issues such as installation failures may occur.

Docker images are available on [Docker Hub](https://hub.docker.com/).

#### Quick Start with Docker

**Step 1: Pull the image**

```bash
# Pull from Docker Hub (Rocky 9)
docker pull jetfuls/kickcraft:1.0-rocky9
# or
docker pull crpi-g7nxbvns4i9rnvaf.cn-hangzhou.personal.cr.aliyuncs.com/jetfuls/kickcraft:1.0-rocky9
# Rocky 8
docker pull jetfuls/kickcraft:1.0-rocky8
# or
docker pull crpi-g7nxbvns4i9rnvaf.cn-hangzhou.personal.cr.aliyuncs.com/jetfuls/kickcraft:1.0-rocky8

# Rocky 10
docker pull jetfuls/kickcraft:1.0-rocky10
# or
docker pull crpi-g7nxbvns4i9rnvaf.cn-hangzhou.personal.cr.aliyuncs.com/jetfuls/kickcraft:1.0-rocky10 
```

**Step 2: Run the container**

```bash
# KickCraft Rocky 9
docker run -itd -p 8080:8080 --name kickcraft jetfuls/kickcraft:1.0-rocky9
# or
docker run -itd -p 8080:8080 --name kickcraft crpi-g7nxbvns4i9rnvaf.cn-hangzhou.personal.cr.aliyuncs.com/jetfuls/kickcraft:1.0-rocky9

# KickCraft Rocky 8
docker run -itd -p 8080:8080 --name kickcraft jetfuls/kickcraft:1.0-rocky8
# or
docker run -itd -p 8080:8080 --name kickcraft crpi-g7nxbvns4i9rnvaf.cn-hangzhou.personal.cr.aliyuncs.com/jetfuls/kickcraft:1.0-rocky8
# KickCraft Rocky 10
docker run -itd -p 8080:8080 --name kickcraft jetfuls/kickcraft:1.0-rocky10
# or
docker run -itd -p 8080:8080 --name kickcraft crpi-g7nxbvns4i9rnvaf.cn-hangzhou.personal.cr.aliyuncs.com/jetfuls/kickcraft:1.0-rocky10
```

**Step 3: Access the service**

Once the container is running, you can access:

- Web UI → [http://localhost:8080](http://localhost:8080)
- API Docs → [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

**Step 4: Manage the container**

```bash
# View container logs
docker logs kickcraft

# Follow logs in real-time
docker logs -f kickcraft

# Stop the container
docker stop kickcraft

# Start the container again
docker start kickcraft

# Remove the container
docker rm -f kickcraft
```

### Building the Docker image

You can build a docker image locally with the following commands:

```bash
# Build for Rocky 9 (default)
make docker-build ROCKY_VERSION=9

# Build for Rocky 8
make docker-build ROCKY_VERSION=8

# Build for Rocky 10
make docker-build ROCKY_VERSION=10

# Run
make docker-run ROCKY_VERSION=9
```

Notes:
- `ROCKY_VERSION`: Rocky Linux version (8, 9, or 10)

### Docker Compose

If you are using `docker-compose.yml`, you can start the service stack with:

```bash
# Start (Rocky 9, auto-builds the image if not present)
make compose-up

# Or specify a different compose file
make compose-up COMPOSE_FILE=docker-compose-rocky8.yml

# View logs
make compose-logs

# Stop
make compose-down
```

### Health Check

To check the health of your KickCraft instance, use the following command:

```bash
curl -sf http://localhost:8080/health && echo OK || echo FAIL
```

## Requirements

- Go 1.24+
- Linux (Ubuntu, Rocky, or CentOS recommended)
- Tools: `xorriso`, `7z` or `p7zip`

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/distros` | GET | List supported distributions |
| `/api/distros/sources` | GET | List ISO download sources |
| `/api/config/parse` | POST | Parse ks.cfg text |
| `/api/config/validate` | POST | Validate configuration |
| `/api/config/generate` | POST | Generate ks.cfg from config |
| `/api/templates` | GET/POST/DELETE | Template management |
| `/api/iso/generate` | POST | Generate ISO |
| `/api/iso/status/:id` | GET | Check build status |

## Documentation

- [Kickstart Documentation](https://pykickstart.readthedocs.io/)
- [pykickstart Commands](https://pykickstart.readthedocs.io/en/latest/kickstart-docs.html)

### Supported Platforms

Considering that this project is designed for Rocky Linux/CentOS:

- In a non-Docker environment, it can only run on a Linux host.
- In a Docker environment, it can run on any operating system.

### FAQ

- **Blank UI / 404?** → Check `STATIC_DIR` path
- **Health check fails?** → Inspect logs: `docker logs kickcraft`
- **ISO build fails?** → Fix config validation errors, then check network

## License

MIT License
