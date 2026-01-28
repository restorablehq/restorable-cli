---
title: Installation
type: docs
weight: 1
prev: docs/getting-started
next: docs/commands
---

This guide covers different ways to install Restorable CLI on your system.

## Prerequisites

Before installing Restorable CLI, ensure you have:

- **Docker** - Required for running ephemeral database containers
- **Go 1.21 or later** - Only required if building from source

### Verifying Prerequisites

```bash
# Check Docker is running
docker info
# Should display Docker system information
```

## Installation Methods

### Method 1: Install Script (Recommended)

The easiest way to install Restorable CLI is using the official install script:

```bash
curl -fsSL https://get.restorable.io | sh
```

This script:
- Detects your operating system and architecture
- Downloads the appropriate binary
- Installs it to `/usr/local/bin`
- Verifies the installation

### Method 2: Download Pre-built Binary

Pre-built binaries are available for major platforms on the releases page.

{{< tabs items="Linux,macOS,macOS (Silicon)" >}}

  {{< tab >}}
  ```bash
  curl -LO https://github.com/your-org/restorable-cli/releases/latest/download/restorable-linux-amd64
  chmod +x restorable-linux-amd64
  sudo mv restorable-linux-amd64 /usr/local/bin/restorable
  ```
  {{< /tab >}}

  {{< tab >}}
  ```bash
  curl -LO https://github.com/your-org/restorable-cli/releases/latest/download/restorable-darwin-amd64
  chmod +x restorable-darwin-amd64
  sudo mv restorable-darwin-amd64 /usr/local/bin/restorable
  ```
  {{< /tab >}}

  {{< tab >}}
  ```bash
  curl -LO https://github.com/your-org/restorable-cli/releases/latest/download/restorable-darwin-arm64
  chmod +x restorable-darwin-arm64
  sudo mv restorable-darwin-arm64 /usr/local/bin/restorable
  ```
  {{< /tab >}}

{{< /tabs >}}

### Method 3: Build from Source

For development or customization, build from source:

```bash
# Prerequisites: Go 1.21+ and Git

# Clone the repository
git clone https://github.com/your-org/restorable-cli.git
cd restorable-cli

# Build the binary
go build -o restorable ./cmd/restorable

# Move to a directory in your PATH
sudo mv restorable /usr/local/bin/
```

### Method 4: Using Go Install

If you have Go configured with `GOBIN` in your PATH:

```bash
go install github.com/your-org/restorable-cli/cmd/restorable@latest
```

## Verifying Installation

After installation, verify the CLI is working:

```bash
restorable version
# Output: restorable version 0.1.0
```

## Docker Setup

Restorable uses Docker to run ephemeral database containers. Ensure Docker is properly configured:

### Linux

```bash
# Start Docker daemon
sudo systemctl start docker

# Add your user to the docker group (optional, avoids sudo)
sudo usermod -aG docker $USER
# Log out and back in for group changes to take effect
```

### macOS

Install Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop) and ensure it's running.

### Verify Docker Access

```bash
# Test Docker without sudo
docker run hello-world
```

## Directory Structure

After running `restorable init`, the following directory structure is created:

```
~/.restorable/
├── config.yaml         # Main configuration file
├── keys/
│   ├── signing.key     # Ed25519 private key for signing reports
│   ├── signing.pub     # Ed25519 public key for verification
│   └── backup.key      # Age private key (if using encryption)
├── reports/            # Generated verification reports
└── schemas/            # Baseline schemas for comparison
```

## Uninstallation

To remove Restorable CLI:

```bash
# Remove the binary
sudo rm /usr/local/bin/restorable

# Optionally, remove configuration and data
rm -rf ~/.restorable
```

## Next Steps

- [Getting Started](getting-started.md) - Run your first backup verification
- [Configuration](configuration.md) - Configure Restorable for your environment
