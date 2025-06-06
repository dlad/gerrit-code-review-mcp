# Gerrit Code Review MCP

A Model Context Protocol (MCP) server that provides tools for interacting with Gerrit Code Review systems.

## Features

- Connect to Gerrit instances with multiple authentication methods
- Retrieve Gerrit change information and patches
- MCP-compatible tool interface for integration with AI assistants

## Quick Start

### Using Docker (Recommended)

1. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your Gerrit credentials:
   ```env
   GERRIT_BASE_URL=https://your-gerrit-instance.com
   GERRIT_USERNAME=your-username
   GERRIT_PASSWORD=your-password
   ```

3. Build and run with Docker Compose:
   ```bash
   docker-compose up --build
   ```

### Manual Build

1. Ensure Go 1.24.3+ is installed
2. Build the binary:
   ```bash
   go build cmd/gerrit-code-review-mcp.go
   ```
3. Set environment variables and run:
   ```bash
   export GERRIT_BASE_URL="https://your-gerrit-instance.com"
   export GERRIT_USERNAME="your-username"
   export GERRIT_PASSWORD="your-password"
   ./gerrit-code-review-mcp
   ```

## Configuration

The application requires the following environment variables:

- `GERRIT_BASE_URL`: Base URL of your Gerrit instance
- `GERRIT_USERNAME`: Your Gerrit username (optional for anonymous access)
- `GERRIT_PASSWORD`: Your Gerrit password or HTTP password (optional for anonymous access)
