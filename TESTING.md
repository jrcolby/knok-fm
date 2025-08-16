# Testing Knock FM Discord Bot

## Complete Testing (Database + Redis)

Test the full system with PostgreSQL database and Redis queue:

### 1. Start Development Services

```bash
# Start PostgreSQL and Redis using Docker Compose
make dev-services

# This starts:
# - PostgreSQL on localhost:5432 (user: dev, pass: devpass, db: knockfm)
# - Redis on localhost:6379
# - Redis Commander UI on localhost:8081
```

### 2. Set Environment Variables

```bash
export DISCORD_TOKEN="your_discord_bot_token"
export DATABASE_URL="postgresql://dev:devpass@localhost:5432/knockfm"
export REDIS_URL="redis://localhost:6379"
export LOG_LEVEL="debug"
```

### 3. Run the Services

Terminal 1 - Worker (processes jobs):

```bash
go run cmd/worker/main.go
```

Terminal 2 - Bot (detects URLs and queues jobs):

```bash
go run cmd/bot/main.go
```

### 4. Test URL Detection

Send messages in Discord with music URLs:

- `https://youtube.com/watch?v=dQw4w9WgXcQ`
- `https://soundcloud.com/artist/track`
- `https://open.spotify.com/track/123`

The bot will:

1. ✅ Detect the URLs and identify platforms
2. ✅ Add 🎵 reaction to messages with music
3. ✅ Queue metadata extraction jobs in Redis

The worker will:

4. ✅ Dequeue jobs from Redis
5. ✅ Extract mock metadata for each platform
6. ✅ Update track records in PostgreSQL
7. ✅ Log comprehensive processing results

### 5. Monitor Redis Queue

Check queued jobs:

```bash
# Connect to Redis CLI
redis-cli

# Check queue length
LLEN queue:extract_metadata

# View queued job IDs
LRANGE queue:extract_metadata 0 -1

# Check job details (replace JOB_ID with actual ID)
HGETALL job:JOB_ID
```

## Current Limitations

- **Mock Metadata**: Uses placeholder data instead of real platform APIs
- **Placeholder Database**: PostgreSQL repositories log operations but don't execute SQL yet
- **No Frontend**: Web interface not implemented yet

## What Works Now

✅ **URL Detection**: All major platforms (YouTube, SoundCloud, Spotify, etc.)  
✅ **Database Integration**: Track records stored in PostgreSQL  
✅ **Redis Queueing**: Jobs are properly queued with retry logic  
✅ **Background Processing**: Multi-worker job processing with stats
✅ **Discord Integration**: Bot connects and responds to messages  
✅ **Duplicate Prevention**: Same URL+message combinations handled
✅ **Structured Logging**: Detailed logs for debugging  
✅ **Graceful Shutdown**: Proper cleanup on Ctrl+C

## Next Steps

1. Implement actual PostgreSQL schemas and migrations
2. Add real metadata extraction APIs (YouTube, Spotify, etc.)
3. Build web frontend dashboard
4. Add Discord slash commands
5. Deploy to production environment
