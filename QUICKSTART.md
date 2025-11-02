# Quick Start Guide

Get Heimdall up and running in 5 minutes!

## Step 1: Discord Bot Setup

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Click "New Application" and give it a name
3. Go to "Bot" section and click "Add Bot"
4. Copy the bot token (you'll need this for config.yaml)
5. Enable these Privileged Gateway Intents:
   - Server Members Intent
   - Message Content Intent
6. Go to "OAuth2" > "URL Generator"
7. Select scopes: `bot` and `applications.commands`
8. Select permissions:
   - Manage Roles
   - Send Messages
   - Read Message History
   - Use Slash Commands
9. Copy the generated URL and use it to invite the bot to your server

## Step 2: Get Discord IDs

Enable Developer Mode in Discord (User Settings > Advanced > Developer Mode)

Then right-click to copy IDs:
- Server ID: Right-click your server icon → Copy ID
- Role IDs: Server Settings > Roles > Right-click role → Copy ID

## Step 3: Configure Email (Gmail Example)

1. Go to [Google Account Settings](https://myaccount.google.com/)
2. Navigate to Security
3. Enable 2-Factor Authentication if not already enabled
4. Go to "App passwords"
5. Generate a new app password for "Mail"
6. Copy this password (you'll use it in config.yaml)

## Step 4: Configure Heimdall

1. Copy the example config:
   ```bash
   cp config.yaml.example config.yaml
   ```

2. Edit `config.yaml`:
   ```yaml
   discord:
     token: "YOUR_BOT_TOKEN_FROM_STEP_1"
     guild_id: "YOUR_SERVER_ID_FROM_STEP_2"
     admin_role: "ADMIN_ROLE_ID_FROM_STEP_2"

   email:
     smtp_host: "smtp.gmail.com"
     smtp_port: 587
     smtp_username: "youremail@gmail.com"
     smtp_password: "APP_PASSWORD_FROM_STEP_3"
     from_address: "youremail@gmail.com"
     from_name: "Heimdall Bot"

   server:
     port: 8080
     base_url: "http://localhost:8080"  # Change to your domain in production

   approved_domains:
     - "yourcompany.com"

   teams:
     "Engineering": "ENGINEERING_ROLE_ID"
     "Product": "PRODUCT_ROLE_ID"
   ```

## Step 5: Run Heimdall

### Option A: Direct Run (for testing)
```bash
go run .
```

### Option B: Build and Run
```bash
./setup.sh
./heimdall
```

### Option C: Docker
```bash
docker-compose up -d
```

## Step 6: Test It!

1. Join your Discord server with a test account (or have a friend join)
2. The bot should DM you asking for an email
3. Reply with an email from an approved domain
4. Check your email for the verification link
5. Click the link and select a team
6. You should be verified and have the role assigned!

## Troubleshooting Quick Fixes

**Bot doesn't DM new members:**
- Make sure bot has permission to send DMs
- Check bot is online in Discord
- Verify bot has "Send Messages" permission

**Email not received:**
- Check spam folder
- Verify SMTP credentials are correct
- Make sure 2FA is enabled for Gmail
- Use app password, not regular password

**Verification link doesn't work:**
- If testing locally, use ngrok: `ngrok http 8080`
- Update `base_url` in config.yaml to your ngrok URL
- Restart the bot after config changes

**Role not assigned:**
- Bot's role must be higher than roles it assigns
- Bot needs "Manage Roles" permission
- Verify role IDs are correct in config.yaml

## Production Deployment

For production, you should:

1. Use a proper domain with HTTPS
2. Set up a reverse proxy (nginx/Caddy)
3. Use environment variables for secrets
4. Set up systemd service or Docker
5. Configure firewall rules
6. Set up regular database backups

See README.md for detailed production deployment instructions.

## Need Help?

Check the full README.md for comprehensive documentation and troubleshooting guides.
