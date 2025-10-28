# 🚀 Knok FM Production Deployment Guide

Complete guide for deploying Knok FM to a Digital Ocean droplet with Docker, Caddy, and auto-SSL.

## Quick Start Checklist

- [ ] Digital Ocean droplet created (1GB RAM recommended)
- [ ] Domain DNS configured (knok-fm.com, api.knok-fm.com)
- [ ] Discord bot token obtained
- [ ] `.env.prod` file configured
- [ ] Services deployed and running
- [ ] Database seeded from Discord channel
- [ ] SSL certificates auto-provisioned
- [ ] Frontend and API verified working

---

## Phase 1: Server Setup

### 1.1 Create Digital Ocean Droplet

1. Log in to Digital Ocean
2. Create Droplet:
   - **Image**: Ubuntu 24.04 LTS
   - **Plan**: Basic $6/mo (1GB RAM / 1 vCPU / 25GB SSD)
   - **Region**: Choose closest to your users
   - **Authentication**: Add your SSH key
   - **Hostname**: knok-fm-prod

### 1.2 Initial Configuration

```bash
# SSH as root
ssh root@YOUR_DROPLET_IP

# Update system
apt update && apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh
apt install docker-compose-plugin -y

# Create app user
adduser knokfm --disabled-password --gecos ""
usermod -aG docker knokfm

# Setup firewall
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable

# Disable root SSH
sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
systemctl restart sshd

# Copy SSH key to knokfm user (from your local machine)
# ssh-copy-id knokfm@YOUR_DROPLET_IP
```

---

## Phase 2: DNS Configuration (Porkbun)

Add these A records in Porkbun:

| Type | Host | Answer          | TTL |
|------|------|-----------------|-----|
| A    | @    | YOUR_DROPLET_IP | 600 |
| A    | www  | YOUR_DROPLET_IP | 600 |
| A    | api  | YOUR_DROPLET_IP | 600 |

Wait 5-10 minutes for DNS propagation.

Verify:
```bash
dig knok-fm.com +short
dig api.knok-fm.com +short
```

---

## Phase 3: Deploy Application

### 3.1 Clone Repository

```bash
# SSH as knokfm user
ssh knokfm@YOUR_DROPLET_IP

cd ~
git clone https://github.com/YOUR_USERNAME/knok-fm.git
cd knok-fm
```

### 3.2 Configure Environment

```bash
# Copy template
cp .env.prod.example .env.prod

# Edit with production values
nano .env.prod
```

**Update these values:**
- `POSTGRES_PASSWORD`: Generate with `openssl rand -base64 32`
- `DATABASE_URL`: Use same password as POSTGRES_PASSWORD
- `DISCORD_TOKEN`: Your actual Discord bot token
- `ADMIN_API_KEY`: Generate with `openssl rand -hex 32`
- `VITE_API_BASE_URL`: Set to `https://api.knok-fm.com`

### 3.3 Build Frontend with Production URL

```bash
# Set API URL for frontend build
echo "VITE_API_BASE_URL=https://api.knok-fm.com" > web/.env
```

### 3.4 Start Services

```bash
# Build all services (takes 5-10 minutes)
docker compose -f docker-compose.prod.yml build

# Start everything
docker compose -f docker-compose.prod.yml up -d

# Check status
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f
```

---

## Phase 4: Database Seeding

### Seed from Discord Channel

```bash
# Run seeder inside API container
docker compose -f docker-compose.prod.yml exec api ./seeder \
  -channel YOUR_CHANNEL_ID \
  -guild YOUR_GUILD_ID \
  -limit 1000

# Verify data
docker compose -f docker-compose.prod.yml exec postgres \
  psql -U knokfm -d knokfm -c "SELECT COUNT(*) FROM knoks;"
```

---

## Phase 5: Verification

### Test Endpoints

```bash
# API health
curl https://api.knok-fm.com/health
# Should return: {"status":"ok"}

# Get knoks
curl https://api.knok-fm.com/api/v1/knoks | jq

# Frontend
curl -I https://knok-fm.com
# Should return 200 OK
```

### Browser Testing

1. Visit https://knok-fm.com
2. Verify SSL padlock icon
3. Test search functionality
4. Test infinite scroll
5. Check browser console for errors

### Discord Bot Test

1. Post a music URL in Discord
2. Bot should react with 🎵
3. URL should appear on website

---

## Maintenance

### View Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f bot
docker compose -f docker-compose.prod.yml logs -f api
```

### Update Application

```bash
cd ~/knok-fm
./scripts/deploy.sh

# Or manually:
git pull origin main
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d
```

### Backup Database

```bash
# Create backup
docker compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U knokfm knokfm > backup_$(date +%Y%m%d).sql

# Restore backup
cat backup.sql | docker compose -f docker-compose.prod.yml exec -T postgres \
  psql -U knokfm -d knokfm
```

### Restart Services

```bash
# Restart all
docker compose -f docker-compose.prod.yml restart

# Restart one service
docker compose -f docker-compose.prod.yml restart api
```

---

## Troubleshooting

### Services Not Starting

```bash
# Check logs
docker compose -f docker-compose.prod.yml logs

# Check status
docker compose -f docker-compose.prod.yml ps

# Recreate
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d
```

### SSL Issues

Caddy provisions SSL automatically. If failing:
1. Verify DNS points to droplet
2. Check ports 80/443 are open
3. View Caddy logs: `docker compose -f docker-compose.prod.yml logs caddy`

### Bot Not Responding

```bash
# Check bot logs
docker compose -f docker-compose.prod.yml logs -f bot

# Restart bot
docker compose -f docker-compose.prod.yml restart bot
```

### Frontend API Errors

1. Verify API is accessible: `curl https://api.knok-fm.com/health`
2. Check CORS in Caddyfile
3. Rebuild frontend if API URL changed

### Out of Disk Space

```bash
# Clean Docker
docker system prune -a

# Check disk
df -h
```

---

## Monitoring

```bash
# Resource usage
docker stats

# Disk space
df -h

# Memory
free -h
```

---

## Security Checklist

- ✅ SSL enabled (Caddy auto-SSL)
- ✅ Strong passwords for DB and Admin API
- ✅ Firewall configured (UFW)
- ✅ Root SSH disabled
- ✅ Services run as non-root
- ✅ .env.prod not in git
- ⚠️ Consider enabling DO backups ($1.20/mo)
- ⚠️ Consider adding fail2ban

---

## Cost

- **Droplet**: $6/month
- **Domain**: ~$10/year (already purchased)
- **SSL**: Free (Let's Encrypt)
- **Backups**: Optional $1.20/month

**Total**: ~$6-7/month

---

## Quick Reference

```bash
# Deploy updates
cd ~/knok-fm && ./scripts/deploy.sh

# View logs
docker compose -f docker-compose.prod.yml logs -f

# Restart service
docker compose -f docker-compose.prod.yml restart SERVICE_NAME

# Backup database
docker compose -f docker-compose.prod.yml exec postgres pg_dump -U knokfm knokfm > backup.sql

# SSH into server
ssh knokfm@YOUR_DROPLET_IP
```

---

**Last Updated**: January 2025
