# Knok FM - Discord Music Bot

A Discord music bot built with Go that automatically detects and tracks music URLs shared in your server. Features advanced URL detection, metadata extraction, and a web dashboard for browsing shared music.

## Features

- **Smart URL Detection**: Detects music URLs in various formats including markdown links, mobile share URLs, and Discord-suppressed embeds
- **Multi-Platform Support**: YouTube, Spotify, SoundCloud, Apple Music, Bandcamp, Tidal, Deezer, and more
- **Mobile Link Support**: Handles platform-specific short URLs (youtu.be, link.tospotify.com, on.soundcloud.com)
- **Automatic Metadata Extraction**: Retrieves track titles, descriptions, and images
- **URL Normalization**: Removes tracking parameters for clean deduplication
- **Flexible Configuration**: Per-server settings for unknown platform handling (permissive/strict modes)
- **Web Dashboard**: Browse and search music shared in your servers
- **REST API**: Full API for integrations and custom frontends

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL
- Redis
- Discord Bot Token

### Setup

1. **Clone the repository**

   ```bash
   git clone https://github.com/yourusername/knock-fm.git
   cd knock-fm
   ```

2. **Install dependencies**

   ```bash
   go mod tidy
   cd web && npm install
   ```

3. **Start development services** (PostgreSQL and Redis)

   ```bash
   make dev-services
   ```

4. **Configure environment**

   ```bash
   cp .env.example .env
   # Edit .env and add your DISCORD_TOKEN
   ```

5. **Run the services**
   ```bash
   # In separate terminals:
   make dev-bot      # Discord bot
   make dev-worker   # Background worker
   make dev-api      # HTTP API server
   make dev-web      # React frontend
   ```

## Configuration

See `.env.example` for all available configuration options.

**Required:**

- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `DISCORD_TOKEN` - Your Discord bot token

**Optional:**

- `UNKNOWN_PLATFORM_MODE` - How to handle unknown platforms (`permissive` or `strict`, default: `permissive`)
- `LOG_LEVEL` - Logging level (`debug`, `info`, `warn`, `error`, default: `info`)
- `PORT` - HTTP server port (default: `8080`)
- `DISCORD_ALLOWED_GUILDS` - Comma-separated Discord server IDs to restrict bot operation (leave empty for all servers)
- `DISCORD_ALLOWED_CHANNELS` - Comma-separated Discord channel IDs to restrict bot listening (leave empty for all channels)

### Discord Server & Channel Restrictions

Control where the bot responds using these optional settings:

**`DISCORD_ALLOWED_GUILDS`** - Restrict which Discord servers (guilds) the bot operates in:

```bash
# Single server
DISCORD_ALLOWED_GUILDS=exampleserverid

# Multiple servers
DISCORD_ALLOWED_GUILDS=exampleserverid1,exampleserverid2

# All servers (default)
DISCORD_ALLOWED_GUILDS=
```

**`DISCORD_ALLOWED_CHANNELS`** - Restrict which channels the bot listens to:

```bash
# Single channel
DISCORD_ALLOWED_CHANNELS=examplechannelid

# Multiple channels
DISCORD_ALLOWED_CHANNELS=DISCORD_ALLOWED_CHANNELS=examplechannelid1,examplechannelid1

# All channels (default)
DISCORD_ALLOWED_CHANNELS=
```

**Note:** You can also configure per-server allowed channels via the database `servers.settings` field.

### Unknown Platform Handling

**Permissive Mode** (default): Accepts all URLs, even from unrecognized platforms
**Strict Mode**: Only processes URLs from known music platforms

Set globally via environment:

```bash
UNKNOWN_PLATFORM_MODE=strict
```

Or override per Discord server via database:

```sql
UPDATE servers
SET settings = jsonb_set(settings, '{unknown_platform_mode}', '"strict"')
WHERE id = 'YOUR_SERVER_ID';
```

## Architecture

Knok FM uses a microservices architecture with three main components:

- **Bot Service** (`cmd/bot`) - Discord bot that detects URLs and queues jobs
- **Worker Service** (`cmd/worker`) - Background workers for metadata extraction
- **API Service** (`cmd/api`) - HTTP REST API for frontend integration
- **Web Frontend** (`web/`) - React dashboard for browsing shared music

See [`CLAUDE.md`](CLAUDE.md) for detailed technical documentation.

## Development

**Make commands:**

- `make dev-services` - Start PostgreSQL and Redis
- `make dev-bot` - Run Discord bot
- `make dev-worker` - Run background worker
- `make dev-api` - Run API server
- `make dev-web` - Run React frontend
- `make test` - Run tests
- `make lint` - Run linting

**Project uses:**

- Podman for containerization (can substitute Docker)
- PostgreSQL for data storage
- Redis for job queues
- Discord.js for bot functionality

Unknown platforms can be handled permissively (accept all) or strictly (reject).

## Contributing

For detailed technical documentation and contribution guidelines, see [`CLAUDE.md`](CLAUDE.md).

## License

[Add your license here]
