# Heimdall Discord Bot - Project Summary

## What You've Got

A complete, production-ready Discord verification bot written in Go that:

✅ **Verifies users via work email** - Users must provide a company email from approved domains
✅ **Sends verification emails** - Automated email with unique verification link
✅ **Web-based team selection** - Beautiful verification page where users select their team
✅ **Automatic role assignment** - Assigns Discord roles based on team selection
✅ **One-to-one relationships** - Each Discord user and email can only be used once
✅ **SQLite database** - Lightweight, file-based storage with full CRUD operations
✅ **Admin commands** - Full suite of slash commands for managing verifications
✅ **Production-ready** - Includes Docker, systemd service, comprehensive docs

## Project Structure

```
heimdall/
├── main.go                  # Entry point - starts bot and web server
├── bot.go                   # Discord bot logic, event handlers, commands
├── config.go                # Configuration file loading
├── database.go              # SQLite operations and schema
├── email.go                 # SMTP email sending
├── webserver.go             # HTTP server with verification UI
├── go.mod & go.sum          # Go dependencies
├── config.yaml.example      # Configuration template
├── Dockerfile               # Container build
├── docker-compose.yml       # Docker orchestration
├── heimdall.service         # Systemd service file
├── setup.sh                 # Automated setup script
├── README.md                # Comprehensive documentation
├── QUICKSTART.md            # 5-minute getting started guide
└── .gitignore              # Git ignore rules

Total: ~600 lines of Go code
```

## Key Features Implemented

### User Flow
1. User joins Discord server
2. Bot sends DM requesting work email
3. User replies with email address
4. Bot validates domain against approved list
5. Bot checks for uniqueness (both Discord ID and email)
6. Bot generates unique verification code
7. Bot sends email with verification link
8. User clicks link and sees verification page
9. User selects their team from dropdown
10. Bot updates database and assigns Discord role
11. User gains full server access

### Admin Features
- `/heimdall-stats` - View verification statistics
- `/heimdall-list` - List all users with status
- `/heimdall-reset @user` - Remove user and allow re-verification
- `/heimdall-domains` - View approved email domains
- `/heimdall-help` - Display help information

### Database Schema
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    discord_id TEXT UNIQUE NOT NULL,
    discord_username TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    verification_code TEXT UNIQUE NOT NULL,
    team_role TEXT,
    verified BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    verified_at DATETIME
);
```

### Security Features
- Domain whitelist validation
- Unique verification codes (64 hex characters)
- One-to-one Discord ID ↔ Email mapping
- Admin-only command restrictions
- HTTPS-ready for production
- No sensitive data in logs

## Technology Stack

- **Language**: Go 1.23
- **Discord Library**: discordgo v0.28.1 (latest)
- **Database**: SQLite3 with go-sqlite3 driver v1.14.24
- **Config**: YAML with gopkg.in/yaml.v3
- **Web Server**: Standard library net/http
- **Email**: Standard library net/smtp

## Deployment Options

### Option 1: Direct Run (Development)
```bash
./setup.sh
./heimdall
```

### Option 2: Systemd Service (Linux Production)
```bash
sudo cp heimdall.service /etc/systemd/system/
sudo systemctl enable --now heimdall
```

### Option 3: Docker (Containerized)
```bash
docker-compose up -d
```

## Configuration Required

Before running, you need to configure in `config.yaml`:

1. **Discord**:
   - Bot token from Discord Developer Portal
   - Guild (server) ID
   - Admin role ID (optional)

2. **Email (SMTP)**:
   - SMTP host and port (e.g., smtp.gmail.com:587)
   - Username and password (use app password for Gmail)
   - From address and name

3. **Web Server**:
   - Port to listen on (default: 8080)
   - Base URL for verification links (must be publicly accessible)

4. **Approved Domains**:
   - List of company email domains to accept

5. **Teams**:
   - Mapping of team names to Discord role IDs

## Getting Started

See `QUICKSTART.md` for a 5-minute setup guide, or `README.md` for comprehensive documentation.

## Next Steps

1. Edit `config.yaml` with your settings
2. Run `./setup.sh` to build
3. Start the bot with `./heimdall`
4. Test with a new member joining your server
5. Deploy to production when ready

## Support & Documentation

- **Quick Start**: Read QUICKSTART.md
- **Full Docs**: Read README.md
- **Issues**: Check troubleshooting section in README
- **Deployment**: See deployment options in README

---

Built with ❤️ in Go. Named after Heimdallr, the Norse god who guards the Bifröst bridge.
