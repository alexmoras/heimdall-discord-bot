# 🛡️ Heimdall - Discord Verification Bot

Heimdall is a Discord bot that verifies new server members through work email authentication. It ensures that only users with approved company email addresses can access your server, and allows them to select their team role during verification.

***This is vibe-coded by Claude Code to quickly solve a problem I had - its probably not a good idea to rely on this in a large server.***

## Features

- ✉️ **Email Verification**: Users verify their identity using work email addresses
- 🏢 **Domain Whitelisting**: Only approved company domains can be used
- 👥 **Team Role Assignment**: Users select their team during verification and receive the appropriate Discord role
- 🔒 **One-to-One Mapping**: Each Discord user and email address can only be used once
- 📊 **Admin Commands**: View statistics, list users, and manage verifications
- 🌐 **Web Interface**: Beautiful verification page for team selection
- 💾 **SQLite Database**: Lightweight, file-based storage
- 🔍 **Status Monitoring**: JSON status endpoint for uptime monitoring
- 🛡️ **GDPR Compliant**: User data deletion commands for privacy compliance

## Prerequisites

- Go 1.23 or higher
- A Discord bot token
- SMTP server credentials (for sending emails)
- A web server accessible from the internet (for verification links)

## Installation

For detailed installation instructions, see [INSTALL.md](INSTALL.md).

**Quick version:**

1. **Clone or download this repository**

2. **Install dependencies**:
```bash
cd heimdall
go mod download
go mod tidy
```

3. **Copy the example configuration**:
```bash
cp config.yaml.example config.yaml
```

4. **Edit `config.yaml`** with your settings (see Configuration section below)

5. **Build the bot**:
```bash
go build -o heimdall
```

6. **Run the bot**:
```bash
./heimdall
```

**Having issues?** See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for solutions to common problems.

## Configuration

Edit `config.yaml` with your specific settings:

### Discord Settings

```yaml
discord:
  token: "YOUR_DISCORD_BOT_TOKEN"      # Get from Discord Developer Portal
  guild_id: "YOUR_GUILD_ID"            # Your server ID
  admin_role: "YOUR_ADMIN_ROLE_ID"     # Role ID for admins (optional)
```

**How to get these values:**
- **Bot Token**: Go to [Discord Developer Portal](https://discord.com/developers/applications), create an application, add a bot, and copy the token
- **Guild ID**: Enable Developer Mode in Discord (User Settings > Advanced), right-click your server, and select "Copy ID"
- **Admin Role ID**: Right-click the admin role in Server Settings > Roles and select "Copy ID"

**⚠️ CRITICAL - Enable Required Intents:**
Before your bot will work, you MUST enable these intents in the Discord Developer Portal:
1. Go to your application > Bot section
2. Scroll to "Privileged Gateway Intents"
3. Enable **"Server Members Intent"** ✅
4. Enable **"Message Content Intent"** ✅
5. Click "Save Changes"

Without these intents, you'll get error: `websocket: close 4014: Disallowed intent(s)`

See [INTENT_FIX.md](INTENT_FIX.md) if you encounter this error.

### Bot Permissions

Your bot needs these permissions:
- Manage Roles
- Send Messages
- Read Message History
- Use Slash Commands
- View Channels

**Invite URL Template**:
```
https://discord.com/api/oauth2/authorize?client_id=YOUR_CLIENT_ID&permissions=268435456&scope=bot%20applications.commands
```

Replace `YOUR_CLIENT_ID` with your bot's client ID from the Discord Developer Portal.

### Email Settings

```yaml
email:
  smtp_host: "smtp.gmail.com"          # Your SMTP server
  smtp_port: 587                        # Usually 587 for TLS
  smtp_username: "your-email@gmail.com"
  smtp_password: "your-app-password"   # Use app-specific password for Gmail
  from_address: "noreply@yourdomain.com"
  from_name: "Heimdall Bot"
```

**Gmail Users**: Create an [App Password](https://support.google.com/accounts/answer/185833) instead of using your regular password.

### Server Settings

```yaml
server:
  port: 8080                           # Port for the web server
  base_url: "https://yourdomain.com"   # Your public URL (no trailing slash)
```

**Important**: The `base_url` must be accessible from the internet. Users will click verification links that point to this URL. Consider using:
- A VPS or cloud server
- Ngrok for testing (e.g., `https://abc123.ngrok.io`)
- Cloudflare Tunnel
- A reverse proxy

### Approved Domains

```yaml
approved_domains:
  - "yourcompany.com"
  - "yourcompany.co.uk"
  - "partner-company.com"
```

Only emails from these domains will be accepted during verification.

### Team Roles

```yaml
teams:
  "Engineering": "ROLE_ID_1"
  "Product": "ROLE_ID_2"
  "Design": "ROLE_ID_3"
```

Map team names to Discord role IDs. Users will select their team during verification and receive the corresponding role.

**How to get Role IDs**: In Discord Server Settings > Roles, right-click a role and select "Copy ID" (Developer Mode must be enabled).

### Members Role (Optional but Recommended)

```yaml
discord:
  members_role: "MEMBERS_ROLE_ID"
```

The Members Role is a base role that gets assigned to ALL verified users, regardless of their team selection. This is useful for:
- Setting base permissions that apply to everyone
- Distinguishing verified from unverified users
- Easier permission management

See [MEMBERS_ROLE.md](MEMBERS_ROLE.md) for detailed setup and usage guide.

## Usage

### For New Members

1. User joins the Discord server
2. Heimdall sends them a DM asking for their work email
3. User replies with their email address (e.g., `john@company.com`)
4. Heimdall validates the domain and sends a verification email
5. User clicks the link in the email
6. User selects their team on the verification page
7. Heimdall assigns the appropriate role and grants server access

### Moderator Commands

All commands are slash commands and are restricted to users with the admin role or Administrator permission:

- `/heimdall-stats` - View verification statistics (total, verified, pending)
- `/heimdall-list` - List all users and their verification status
- `/heimdall-reset @user` - Reset a user's verification (removes from database permanently)
- `/heimdall-domains` - List approved email domains
- `/heimdall-verify @user email team` - Manually verify a user without email flow
  - Example: `/heimdall-verify @JohnDoe email:john@company.com team:Engineering`
  - Use cases: Quick onboarding, users without email access, fixing issues

- `/heimdall-changeteam @user team` - Change a verified user's team
  - Example: `/heimdall-changeteam @JohnDoe team:Product`
  - Removes old team role, assigns new team role, updates database

- `/heimdall-restrict @user reason` - Temporarily restrict a user's access (keeps data)
  - Example: `/heimdall-restrict @JohnDoe reason:"Unpaid subscription"`
  - Use cases: Subscription lapses, temporary suspensions, compliance holds
  - User data preserved, can be quickly restored

- `/heimdall-unrestrict @user` - Remove restrictions from a user
  - Example: `/heimdall-unrestrict @JohnDoe`
  - Instantly restores access using saved data

- `/heimdall-purge user/email` - Permanently delete user data (GDPR compliance)
  - Example: `/heimdall-purge user:@JohnDoe` or `/heimdall-purge email:john@company.com`
  - Complete data removal for GDPR/privacy compliance
  - Can delete by Discord user (autocomplete) or email address

- `/heimdall-help` - Show help information

See [MODERATOR_COMMANDS.md](MODERATOR_COMMANDS.md) for detailed documentation and examples.

### For Regular Users

- `/heimdall-help` - View help and instructions

## Web Server Endpoints

Heimdall runs a web server for verification and monitoring:

### `/verify?code=...`
The verification page where users select their team after clicking the email link.

### `/health`
Simple health check endpoint that returns `OK`. Use for basic uptime monitoring.

### `/status`
Comprehensive JSON status endpoint for monitoring systems. Returns:
- Overall health status
- Bot connection status
- Database statistics (total/verified/pending users)
- Server uptime
- Timestamp

**Example response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h15m30s",
  "uptime_seconds": 8130,
  "bot": {
    "connected": true
  },
  "database": {
    "total_users": 42,
    "verified_users": 38,
    "pending_users": 4
  },
  "timestamp": "2025-11-02T15:30:45Z"
}
```

**Use with monitoring services:**
- UptimeRobot: Monitor for HTTP 200 response
- Pingdom: Check for `"status": "healthy"`
- Better Uptime: Parse JSON to verify bot connectivity
- Custom scripts: Monitor `bot.connected` and `uptime_seconds`

No sensitive data (tokens, emails, usernames) is exposed.

## Database

Heimdall uses SQLite to store user data. The database file (`heimdall.db`) is created automatically in the same directory as the executable.

**Schema**:
- User Discord ID (unique)
- Username
- Email address (unique)
- Verification code
- Team/role
- Verification status
- Timestamps

## Security Considerations

1. **Keep your bot token secret** - Never commit it to version control
2. **Use HTTPS** for your verification URL in production
3. **Regular backups** of `heimdall.db`
4. **Limit admin role** to trusted users only
5. **Use app-specific passwords** for email services

## Deployment

### Using systemd (Linux)

Create `/etc/systemd/system/heimdall.service`:

```ini
[Unit]
Description=Heimdall Discord Bot
After=network.target

[Service]
Type=simple
User=heimdall
WorkingDirectory=/opt/heimdall
ExecStart=/opt/heimdall/heimdall
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Then:
```bash
sudo systemctl daemon-reload
sudo systemctl enable heimdall
sudo systemctl start heimdall
```

### Using Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o heimdall

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/heimdall .
COPY --from=builder /app/config.yaml .
EXPOSE 8080
CMD ["./heimdall"]
```

Build and run:
```bash
docker build -t heimdall .
docker run -d -p 8080:8080 -v $(pwd)/heimdall.db:/root/heimdall.db heimdall
```

## Troubleshooting

### Bot doesn't respond to new members
- Check that the bot has "Send Messages" and DM permissions
- Verify the bot is online and connected
- Check bot logs for errors

### Verification emails not sending
- Verify SMTP credentials are correct
- Check if your email provider requires app-specific passwords
- Ensure port 587 (or your SMTP port) is not blocked
- Check logs for SMTP errors

### Verification link doesn't work
- Ensure `base_url` in config matches your actual domain
- Verify the web server is running on the correct port
- Check if the domain is accessible from the internet
- Look for firewall rules blocking the port

### Roles not being assigned
- Verify the bot has "Manage Roles" permission
- Ensure the bot's role is higher than the roles it's trying to assign
- Check that role IDs in config.yaml are correct

### Database errors
- Ensure the bot has write permissions in its directory
- Check disk space
- Verify SQLite is properly installed

## Development

### Project Structure

```
heimdall/
├── main.go           # Entry point
├── bot.go            # Discord bot logic and event handlers
├── config.go         # Configuration loading
├── database.go       # SQLite database operations
├── email.go          # Email sending functionality
├── webserver.go      # HTTP server for verification pages
├── go.mod            # Go module definition
└── config.yaml       # Configuration file
```

### Running in Development

```bash
# Run without building
go run .

# Run with auto-reload (install air first: go install github.com/cosmtrek/air@latest)
air
```

### Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - feel free to use this bot for your own Discord server.

## Support

If you encounter any issues or have questions:
1. Check the troubleshooting section above
2. Review the logs for error messages
3. Open an issue on GitHub with detailed information

## Acknowledgments

- Built with [DiscordGo](https://github.com/bwmarrin/discordgo)
- Uses SQLite for data persistence
- Inspired by the Norse god Heimdallr, guardian of the Bifröst bridge