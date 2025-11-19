package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session      *discordgo.Session
	config       *Config
	db           *Database
	emailService *EmailService
	ready        chan bool
}

func NewBot(token string, config *Config, db *Database, emailService *EmailService) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		session:      session,
		config:       config,
		db:           db,
		emailService: emailService,
		ready:        make(chan bool, 1),
	}

	// Register event handlers
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onGuildMemberAdd)
	session.AddHandler(bot.onMessageCreate)
	session.AddHandler(bot.onInteractionCreate)

	// Set intents
	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages

	return bot, nil
}

func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return err
	}

	// Wait for bot to be ready
	<-b.ready

	// Register slash commands
	if err := b.registerCommands(); err != nil {
		log.Printf("⚠️  Error registering commands: %v", err)
	} else {
		log.Println("✓ Slash commands registered")
	}

	return nil
}

func (b *Bot) Close() {
	b.session.Close()
}

func (b *Bot) onReady(s *discordgo.Session, event *discordgo.Ready) {
	username := s.State.User.Username
	if s.State.User.Discriminator != "0" {
		username = fmt.Sprintf("%s#%s", s.State.User.Username, s.State.User.Discriminator)
	}
	log.Printf("✓ Connected as: %s", username)
	s.UpdateGameStatus(0, "🛡️ Guarding the gates")

	// Signal that bot is ready
	select {
	case b.ready <- true:
	default:
	}
}

func (b *Bot) onGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	username := m.User.Username
	if m.User.Discriminator != "0" {
		username = fmt.Sprintf("%s#%s", m.User.Username, m.User.Discriminator)
	}

	LogInfo("New member joined: %s (ID: %s)", username, m.User.ID)

	// Check if user already exists and is verified
	user, err := b.db.GetUserByDiscordID(m.User.ID)
	if err == nil && user.Verified {
		// User is already verified, assign their roles
		LogInfo("Restoring roles for returning verified user: %s", username)

		// Assign base members role (if configured)
		if b.config.Discord.MembersRole != "" {
			if err := b.AssignRole(m.User.ID, b.config.Discord.MembersRole); err != nil {
				LogError("Error re-assigning members role to %s: %v", username, err)
			} else {
				LogDebug("Assigned members role to %s", username)
			}
		}

		// Assign team role
		if roleID, exists := b.config.Teams[user.TeamRole]; exists {
			if err := b.AssignRole(m.User.ID, roleID); err != nil {
				LogError("Error re-assigning team role to %s: %v", username, err)
			} else {
				LogDebug("Assigned %s team role to %s", user.TeamRole, username)
			}
		}
		return
	}

	// Send welcome DM
	LogDebug("Sending welcome DM to %s", username)
	channel, err := s.UserChannelCreate(m.User.ID)
	if err != nil {
		LogError("Error creating DM channel for %s: %v", username, err)
		return
	}

	// Use configured welcome message, or fallback to default if not set
	welcomeMsg := b.config.Discord.WelcomeMessage
	if welcomeMsg == "" {
		welcomeMsg = `👋 Welcome to the server!

To gain access, you need to verify your work email address.

Please reply to this message with your work email address (e.g., yourname@company.com).

Your email must be from one of our approved company domains.`
	}

	_, err = s.ChannelMessageSend(channel.ID, welcomeMsg)
	if err != nil {
		LogError("Error sending welcome DM to %s: %v", username, err)
	} else {
		LogDebug("Welcome DM sent to %s", username)
	}
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	// Only process DMs
	channel, err := s.Channel(m.ChannelID)
	if err != nil || channel.Type != discordgo.ChannelTypeDM {
		return
	}

	username := m.Author.Username
	if m.Author.Discriminator != "0" {
		username = fmt.Sprintf("%s#%s", m.Author.Username, m.Author.Discriminator)
	}

	LogDebug("Received DM from %s: %s", username, m.Content)

	// Check if user is already in the verification process
	user, err := b.db.GetUserByDiscordID(m.Author.ID)
	if err == nil && user.Verified {
		LogDebug("User %s is already verified, ignoring DM", username)
		s.ChannelMessageSend(m.ChannelID, "✅ You're already verified!")
		return
	}

	// Check if user has been unverified by moderator
	if err == nil && user.Unverified {
		LogInfo("Blocked unverified user %s from using DM verification", username)
		s.ChannelMessageSend(m.ChannelID, "⚠️ Your access has been temporarily restricted. Please contact a moderator to reactivate your account. You cannot use the automatic verification system.")
		return
	}

	// Validate email format
	email := strings.TrimSpace(strings.ToLower(m.Content))
	if !isValidEmail(email) {
		LogDebug("Invalid email format from %s: %s", username, email)
		s.ChannelMessageSend(m.ChannelID, "❌ That doesn't look like a valid email address. Please try again.")
		return
	}

	LogInfo("Processing verification request from %s with email: %s", username, email)

	// Check if email domain is approved
	if !b.isApprovedDomain(email) {
		LogWarn("Rejected email from unapproved domain: %s (user: %s)", email, username)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Sorry, the domain for %s is not approved. Please use your work email from an approved company domain.", email))
		return
	}

	// Check if email already exists
	exists, err := b.db.EmailExists(email)
	if err != nil {
		LogError("Error checking email existence for %s: %v", email, err)
		s.ChannelMessageSend(m.ChannelID, "❌ An error occurred. Please try again later.")
		return
	}
	if exists {
		LogWarn("Duplicate email registration attempt: %s (user: %s)", email, username)
		s.ChannelMessageSend(m.ChannelID, "❌ This email address is already registered. Each email can only be used once.")
		return
	}

	// Check if Discord ID already exists
	exists, err = b.db.DiscordIDExists(m.Author.ID)
	if err != nil {
		LogError("Error checking Discord ID existence for %s: %v", username, err)
		s.ChannelMessageSend(m.ChannelID, "❌ An error occurred. Please try again later.")
		return
	}
	if exists {
		LogDebug("User %s already has verification in progress", username)
		s.ChannelMessageSend(m.ChannelID, "❌ You've already started the verification process. Please check your email for the verification link.")
		return
	}

	// Generate verification code
	verificationCode, err := generateVerificationCode()
	if err != nil {
		LogError("Error generating verification code for %s: %v", username, err)
		s.ChannelMessageSend(m.ChannelID, "❌ An error occurred. Please try again later.")
		return
	}

	// Create user in database
	err = b.db.CreateUser(m.Author.ID, username, email, verificationCode)
	if err != nil {
		LogError("Error creating user %s in database: %v", username, err)
		s.ChannelMessageSend(m.ChannelID, "❌ An error occurred. Please try again later.")
		return
	}
	LogDebug("Created database entry for %s", username)

	// Send verification email
	err = b.emailService.SendVerificationEmail(email, verificationCode, m.Author.Username)
	if err != nil {
		LogError("Error sending verification email to %s: %v", email, err)
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to send verification email. Please contact an administrator.")
		return
	}

	LogSuccess("Verification email sent to %s (user: %s)", email, username)
	successMsg := fmt.Sprintf("✅ Verification email sent to **%s**!\n\nPlease check your inbox and click the verification link.", email)
	if b.config.Features.EnableTeamSelection {
		successMsg += " You'll be asked to select your team, and then you'll have full access to the server."
	} else {
		successMsg += " Once you verify, you'll have full access to the server."
	}
	s.ChannelMessageSend(m.ChannelID, successMsg)
}

func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "heimdall-stats",
			Description: "View verification statistics (Moderator only)",
		},
		{
			Name:        "heimdall-list",
			Description: "List all users and their verification status (Moderator only)",
		},
		{
			Name:        "heimdall-reset",
			Description: "Reset a user's verification (Moderator only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to reset",
					Required:    true,
				},
			},
		},
		{
			Name:        "heimdall-restrict",
			Description: "Temporarily restrict a user's access (Moderator only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to restrict",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for restriction (optional)",
					Required:    false,
				},
			},
		},
		{
			Name:        "heimdall-unrestrict",
			Description: "Remove restrictions from a user (Moderator only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to unrestrict",
					Required:    true,
				},
			},
		},
		{
			Name:        "heimdall-domains",
			Description: "List approved email domains (Moderator only)",
		},
		{
			Name:        "heimdall-purge",
			Description: "Permanently delete user data (GDPR compliance) (Moderator only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to purge (by Discord account)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "email",
					Description: "The user to purge (by email address)",
					Required:    false,
				},
			},
		},
		{
			Name:        "heimdall-help",
			Description: "Show help information",
		},
	}

	// Add manual verify command - always available but team parameter is optional when feature is disabled
	verifyCommandOptions := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "The user to verify",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "email",
			Description: "User's work email address",
			Required:    true,
		},
	}

	if b.config.Features.EnableTeamSelection {
		verifyCommandOptions = append(verifyCommandOptions, &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "team",
			Description: "Team to assign",
			Required:    true,
		})
	}

	commands = append(commands, &discordgo.ApplicationCommand{
		Name:        "heimdall-verify",
		Description: "Manually verify a user (Moderator only)",
		Options:     verifyCommandOptions,
	})

	// Only register changeteam command if feature is enabled
	if b.config.Features.EnableTeamSelection {
		commands = append(commands, &discordgo.ApplicationCommand{
			Name:        "heimdall-changeteam",
			Description: "Change a verified user's team (Moderator only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to change",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "team",
					Description: "New team to assign",
					Required:    true,
				},
			},
		})
	}

	// Use bulk overwrite to avoid rate limits - this replaces ALL commands in one API call
	log.Printf("Registering %d commands using bulk overwrite...", len(commands))
	registeredCommands, err := b.session.ApplicationCommandBulkOverwrite(b.session.State.User.ID, b.config.Discord.GuildID, commands)
	if err != nil {
		return fmt.Errorf("failed to register commands: %v", err)
	}

	// Log successfully registered commands
	for _, cmd := range registeredCommands {
		log.Printf("✓ %s", cmd.Name)
	}

	return nil
}

func (b *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "heimdall-stats":
		b.handleStats(s, i)
	case "heimdall-list":
		b.handleList(s, i)
	case "heimdall-reset":
		b.handleReset(s, i)
	case "heimdall-verify":
		b.handleManualVerify(s, i)
	case "heimdall-changeteam":
		b.handleChangeTeam(s, i)
	case "heimdall-restrict":
		b.handleRestrict(s, i)
	case "heimdall-unrestrict":
		b.handleUnrestrict(s, i)
	case "heimdall-domains":
		b.handleDomains(s, i)
	case "heimdall-purge":
		b.handlePurge(s, i)
	case "heimdall-help":
		b.handleHelp(s, i)
	}
}

func (b *Bot) handleStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	LogDebug("Moderator %s requested stats", i.Member.User.Username)

	total, verified, pending, err := b.db.GetStats()
	if err != nil {
		LogError("Error getting stats: %v", err)
		b.respondEphemeral(s, i, "❌ Error retrieving statistics.")
		return
	}

	LogInfo("Stats retrieved: total=%d verified=%d pending=%d", total, verified, pending)

	embed := &discordgo.MessageEmbed{
		Title: "📊 Heimdall Statistics",
		Color: 0x667eea,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Total Users",
				Value:  fmt.Sprintf("%d", total),
				Inline: true,
			},
			{
				Name:   "Verified",
				Value:  fmt.Sprintf("%d", verified),
				Inline: true,
			},
			{
				Name:   "Pending",
				Value:  fmt.Sprintf("%d", pending),
				Inline: true,
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	LogDebug("Moderator %s requested user list", i.Member.User.Username)

	users, err := b.db.GetAllUsers()
	if err != nil {
		LogError("Error getting users: %v", err)
		b.respondEphemeral(s, i, "❌ Error retrieving user list.")
		return
	}

	if len(users) == 0 {
		LogDebug("User list empty")
		b.respondEphemeral(s, i, "No users in the database yet.")
		return
	}

	LogInfo("User list retrieved: %d users", len(users))

	var description strings.Builder
	for _, user := range users {
		status := "⏳ Pending"
		if user.Verified {
			status = fmt.Sprintf("✅ Verified (%s)", user.TeamRole)
		} else if user.Unverified {
			status = fmt.Sprintf("⚠️ Unverified (was %s)", user.TeamRole)
		}

		description.WriteString(fmt.Sprintf("**%s**\n└ %s\n└ %s\n\n",
			user.DiscordUsername, user.Email, status))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "👥 User List",
		Description: description.String(),
		Color:       0x667eea,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleReset(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options
	userOption := options[0].UserValue(s)

	LogInfo("Moderator %s attempting to reset user: %s", i.Member.User.Username, userOption.Username)

	user, err := b.db.GetUserByDiscordID(userOption.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			LogDebug("Reset failed - user not found: %s", userOption.Username)
			b.respondEphemeral(s, i, "❌ User not found in the database.")
		} else {
			LogError("Error getting user for reset %s: %v", userOption.Username, err)
			b.respondEphemeral(s, i, "❌ Error retrieving user information.")
		}
		return
	}

	// Remove roles if verified
	if user.Verified {
		// Remove team role
		if roleID, exists := b.config.Teams[user.TeamRole]; exists {
			if err := s.GuildMemberRoleRemove(b.config.Discord.GuildID, userOption.ID, roleID); err != nil {
				LogWarn("Error removing team role from %s during reset: %v", userOption.Username, err)
			} else {
				LogDebug("Removed %s team role from %s", user.TeamRole, userOption.Username)
			}
		}

		// Remove members role (if configured)
		if b.config.Discord.MembersRole != "" {
			if err := s.GuildMemberRoleRemove(b.config.Discord.GuildID, userOption.ID, b.config.Discord.MembersRole); err != nil {
				LogWarn("Error removing members role from %s during reset: %v", userOption.Username, err)
			} else {
				LogDebug("Removed members role from %s", userOption.Username)
			}
		}
	}

	// Delete user from database
	err = b.db.DeleteUser(userOption.ID)
	if err != nil {
		LogError("Error deleting user %s from database: %v", userOption.Username, err)
		b.respondEphemeral(s, i, "❌ Error resetting user.")
		return
	}

	LogSuccess("User %s reset by moderator %s", userOption.Username, i.Member.User.Username)
	b.respondEphemeral(s, i, fmt.Sprintf("✅ Reset verification for <@%s>. They can now start the verification process again.", userOption.ID))

	// Notify the user
	b.SendDM(userOption.ID, "Your verification has been reset by an administrator. Please send me your work email address to start the verification process again.")
}

func (b *Bot) handleDomains(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	var domainList strings.Builder
	for _, domain := range b.config.ApprovedDomains {
		domainList.WriteString(fmt.Sprintf("• %s\n", domain))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📧 Approved Email Domains",
		Description: domainList.String(),
		Color:       0x667eea,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleManualVerify(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options
	userOption := options[0].UserValue(s)
	email := strings.TrimSpace(strings.ToLower(options[1].StringValue()))

	var team string
	var roleID string

	// Get team parameter if team selection is enabled
	if b.config.Features.EnableTeamSelection {
		team = options[2].StringValue()
		LogInfo("Moderator %s attempting manual verify: user=%s email=%s team=%s", i.Member.User.Username, userOption.Username, email, team)
	} else {
		LogInfo("Moderator %s attempting manual verify: user=%s email=%s (no team)", i.Member.User.Username, userOption.Username, email)
	}

	// Validate email format
	if !isValidEmail(email) {
		LogDebug("Invalid email format in manual verify: %s", email)
		b.respondEphemeral(s, i, "❌ Invalid email format.")
		return
	}

	// Check if domain is approved
	if !b.isApprovedDomain(email) {
		LogWarn("Rejected unapproved domain in manual verify: %s (moderator: %s)", email, i.Member.User.Username)
		b.respondEphemeral(s, i, "❌ Domain not approved. Email must be from an approved domain.")
		return
	}

	// Check if team exists (only if feature is enabled)
	if b.config.Features.EnableTeamSelection {
		var exists bool
		roleID, exists = b.config.Teams[team]
		if !exists {
			LogDebug("Invalid team selected in manual verify: %s", team)
			b.respondEphemeral(s, i, fmt.Sprintf("❌ Team '%s' not found.\n\n**Available teams:** %s", team, b.getTeamNames()))
			return
		}
	}

	// Check if user already exists
	existingUser, err := b.db.GetUserByDiscordID(userOption.ID)
	if err == nil && existingUser.Verified {
		LogDebug("User %s already verified, manual verify rejected", userOption.Username)
		if b.config.Features.EnableTeamSelection {
			b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is already verified. Use `/heimdall-changeteam` to change their team.", userOption.ID))
		} else {
			b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is already verified.", userOption.ID))
		}
		return
	}

	// Check if email is already used
	emailExists, err := b.db.EmailExists(email)
	if err != nil {
		LogError("Error checking email existence in manual verify: %v", err)
		b.respondEphemeral(s, i, "❌ Database error occurred.")
		return
	}
	if emailExists {
		LogWarn("Duplicate email in manual verify: %s (moderator: %s)", email, i.Member.User.Username)
		b.respondEphemeral(s, i, "❌ This email address is already registered to another user.")
		return
	}

	// Generate verification code (even though it won't be used for email)
	verificationCode, err := generateVerificationCode()
	if err != nil {
		LogError("Error generating verification code in manual verify: %v", err)
		b.respondEphemeral(s, i, "❌ Error generating verification code.")
		return
	}

	username := userOption.Username
	if userOption.Discriminator != "0" {
		username = fmt.Sprintf("%s#%s", userOption.Username, userOption.Discriminator)
	}

	// Create or update user in database
	if existingUser != nil {
		// User exists but not verified, delete and recreate
		b.db.DeleteUser(userOption.ID)
	}

	err = b.db.CreateUser(userOption.ID, username, email, verificationCode)
	if err != nil {
		LogError("Error creating user in manual verify %s: %v", username, err)
		b.respondEphemeral(s, i, "❌ Error creating user in database.")
		return
	}

	// Mark as verified in database
	if b.config.Features.EnableTeamSelection {
		err = b.db.UpdateUserTeam(userOption.ID, team)
		if err != nil {
			LogError("Error updating user team in manual verify %s: %v", username, err)
			b.respondEphemeral(s, i, "❌ Error updating user verification.")
			return
		}
	} else {
		err = b.db.MarkUserVerified(userOption.ID)
		if err != nil {
			LogError("Error marking user verified in manual verify %s: %v", username, err)
			b.respondEphemeral(s, i, "❌ Error updating user verification.")
			return
		}
	}

	// Assign members role (if configured)
	if b.config.Discord.MembersRole != "" {
		if err := b.AssignRole(userOption.ID, b.config.Discord.MembersRole); err != nil {
			LogError("Error assigning members role in manual verify %s: %v", username, err)
		} else {
			LogDebug("Assigned members role to %s (manual verify)", username)
		}
	}

	// Assign team role (only if feature is enabled)
	if b.config.Features.EnableTeamSelection {
		if err := b.AssignRole(userOption.ID, roleID); err != nil {
			LogError("Error assigning team role in manual verify %s: %v", username, err)
			b.respondEphemeral(s, i, "⚠️ User verified in database but failed to assign Discord role. Please assign manually.")
			return
		}
		LogDebug("Assigned %s team role to %s (manual verify)", team, username)
	}

	// Send success message
	if b.config.Features.EnableTeamSelection {
		LogSuccess("Manual verification: %s verified by %s (team: %s, email: %s)", username, i.Member.User.Username, team, email)
		b.respondEphemeral(s, i, fmt.Sprintf("✅ Successfully verified <@%s> with email `%s` and assigned to **%s** team.", userOption.ID, email, team))
		b.SendDM(userOption.ID, fmt.Sprintf("✅ You have been manually verified by a moderator! Welcome to the **%s** team. You now have access to the server.", team))
	} else {
		LogSuccess("Manual verification: %s verified by %s (email: %s)", username, i.Member.User.Username, email)
		b.respondEphemeral(s, i, fmt.Sprintf("✅ Successfully verified <@%s> with email `%s`.", userOption.ID, email))
		b.SendDM(userOption.ID, "✅ You have been manually verified by a moderator! You now have access to the server.")
	}
}

func (b *Bot) handleChangeTeam(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options
	userOption := options[0].UserValue(s)
	newTeam := options[1].StringValue()

	LogInfo("Moderator %s attempting team change: user=%s newTeam=%s", i.Member.User.Username, userOption.Username, newTeam)

	// Check if new team exists
	newRoleID, exists := b.config.Teams[newTeam]
	if !exists {
		LogDebug("Invalid team in change team: %s", newTeam)
		b.respondEphemeral(s, i, fmt.Sprintf("❌ Team '%s' not found.\n\n**Available teams:** %s", newTeam, b.getTeamNames()))
		return
	}

	// Get user from database
	user, err := b.db.GetUserByDiscordID(userOption.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			LogDebug("User not found in change team: %s", userOption.Username)
			b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is not verified. Use `/heimdall-verify` to verify them first.", userOption.ID))
		} else {
			LogError("Error getting user in change team %s: %v", userOption.Username, err)
			b.respondEphemeral(s, i, "❌ Database error occurred.")
		}
		return
	}

	if !user.Verified {
		LogDebug("Unverified user in change team: %s", userOption.Username)
		b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is not verified yet. Use `/heimdall-verify` to verify them first.", userOption.ID))
		return
	}

	oldTeam := user.TeamRole

	// Check if already on this team
	if oldTeam == newTeam {
		LogDebug("User %s already on team %s", userOption.Username, newTeam)
		b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is already on the **%s** team.", userOption.ID, newTeam))
		return
	}

	// Remove old team role
	if oldRoleID, exists := b.config.Teams[oldTeam]; exists {
		if err := s.GuildMemberRoleRemove(b.config.Discord.GuildID, userOption.ID, oldRoleID); err != nil {
			LogWarn("Error removing old team role from %s: %v", userOption.Username, err)
		} else {
			LogDebug("Removed %s team role from %s", oldTeam, userOption.Username)
		}
	}

	// Assign new team role
	if err := b.AssignRole(userOption.ID, newRoleID); err != nil {
		LogError("Error assigning new team role to %s: %v", userOption.Username, err)
		b.respondEphemeral(s, i, "⚠️ Failed to assign new Discord role. Please assign manually.")
		return
	}
	LogDebug("Assigned %s team role to %s", newTeam, userOption.Username)

	// Update database
	err = b.db.UpdateUserTeam(userOption.ID, newTeam)
	if err != nil {
		LogError("Error updating user team in database %s: %v", userOption.Username, err)
		b.respondEphemeral(s, i, "⚠️ Role changed in Discord but database update failed.")
		return
	}

	// Send success message
	LogSuccess("Team change: %s moved from %s to %s by %s", userOption.Username, oldTeam, newTeam, i.Member.User.Username)
	b.respondEphemeral(s, i, fmt.Sprintf("✅ Changed <@%s> from **%s** to **%s** team.", userOption.ID, oldTeam, newTeam))

	// Send DM to user
	b.SendDM(userOption.ID, fmt.Sprintf("📝 Your team has been changed from **%s** to **%s** by a moderator.", oldTeam, newTeam))
}

func (b *Bot) handleRestrict(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options
	userOption := options[0].UserValue(s)

	var reason string
	if len(options) > 1 {
		reason = options[1].StringValue()
	}

	// Get user from database
	user, err := b.db.GetUserByDiscordID(userOption.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is not in the system.", userOption.ID))
		} else {
			log.Printf("Error getting user: %v", err)
			b.respondEphemeral(s, i, "❌ Database error occurred.")
		}
		return
	}

	if !user.Verified {
		b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is not verified.", userOption.ID))
		return
	}

	if user.Unverified {
		b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is already restricted.", userOption.ID))
		return
	}

	// Remove team role
	if roleID, exists := b.config.Teams[user.TeamRole]; exists {
		if err := s.GuildMemberRoleRemove(b.config.Discord.GuildID, userOption.ID, roleID); err != nil {
			log.Printf("Error removing team role: %v", err)
		}
	}

	// Remove members role (if configured)
	if b.config.Discord.MembersRole != "" {
		if err := s.GuildMemberRoleRemove(b.config.Discord.GuildID, userOption.ID, b.config.Discord.MembersRole); err != nil {
			log.Printf("Error removing members role: %v", err)
		}
	}

	// Mark as unverified in database
	if err := b.db.UnverifyUser(userOption.ID); err != nil {
		log.Printf("Error restricting user: %v", err)
		b.respondEphemeral(s, i, "❌ Failed to update database.")
		return
	}

	// Send success message to moderator
	if reason != "" {
		b.respondEphemeral(s, i, fmt.Sprintf("✅ Restricted <@%s>.\n**Reason:** %s\n\nUser has been notified and their roles removed. Use `/heimdall-unrestrict` to restore access.", userOption.ID, reason))
	} else {
		b.respondEphemeral(s, i, fmt.Sprintf("✅ Restricted <@%s>.\n\nUser has been notified and their roles removed. Use `/heimdall-unrestrict` to restore access.", userOption.ID))
	}

	// Send DM to user
	dmMessage := "⚠️ Your server access has been temporarily restricted by a moderator."
	if reason != "" {
		dmMessage += fmt.Sprintf("\n**Reason:** %s", reason)
	}
	dmMessage += "\n\nYou cannot use the automatic verification system. Please contact a moderator to restore your account."
	b.SendDM(userOption.ID, dmMessage)
}

func (b *Bot) handleUnrestrict(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options
	userOption := options[0].UserValue(s)

	// Get user from database
	user, err := b.db.GetUserByDiscordID(userOption.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is not in the system.", userOption.ID))
		} else {
			log.Printf("Error getting user: %v", err)
			b.respondEphemeral(s, i, "❌ Database error occurred.")
		}
		return
	}

	if !user.Unverified {
		b.respondEphemeral(s, i, fmt.Sprintf("❌ <@%s> is not restricted. Use `/heimdall-verify` for new users or `/heimdall-changeteam` to change teams.", userOption.ID))
		return
	}

	// Assign members role (if configured)
	if b.config.Discord.MembersRole != "" {
		if err := b.AssignRole(userOption.ID, b.config.Discord.MembersRole); err != nil {
			log.Printf("Error assigning members role: %v", err)
		}
	}

	// Assign team role
	if roleID, exists := b.config.Teams[user.TeamRole]; exists {
		if err := b.AssignRole(userOption.ID, roleID); err != nil {
			log.Printf("Error assigning team role: %v", err)
			b.respondEphemeral(s, i, "⚠️ Failed to assign Discord roles. Please assign manually.")
			return
		}
	}

	// Mark as unrestricted in database
	if err := b.db.ReverifyUser(userOption.ID); err != nil {
		log.Printf("Error unrestricting user: %v", err)
		b.respondEphemeral(s, i, "❌ Failed to update database.")
		return
	}

	// Send success message
	b.respondEphemeral(s, i, fmt.Sprintf("✅ Removed restrictions from <@%s> on the **%s** team. Their access has been restored.", userOption.ID, user.TeamRole))

	// Send DM to user
	b.SendDM(userOption.ID, fmt.Sprintf("✅ Your server access has been restored by a moderator! Welcome back to the **%s** team.", user.TeamRole))
}

func (b *Bot) handlePurge(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.isAdmin(i.Member) {
		b.respondEphemeral(s, i, "❌ You don't have permission to use this command.")
		return
	}

	options := i.ApplicationCommandData().Options

	// Check that at least one option is provided
	if len(options) == 0 {
		b.respondEphemeral(s, i, "❌ Please provide either a user or an email address.")
		return
	}

	// Try to find user by the provided option
	var user *User
	var err error
	var identifierType string
	var identifier string

	// Check which option was provided
	for _, opt := range options {
		if opt.Name == "user" {
			userOption := opt.UserValue(s)
			identifier = userOption.Username
			user, err = b.db.GetUserByDiscordID(userOption.ID)
			identifierType = "Discord user"
			LogInfo("Moderator %s attempting purge for Discord user: %s (ID: %s)", i.Member.User.Username, userOption.Username, userOption.ID)
			break
		} else if opt.Name == "email" {
			identifier = opt.StringValue()
			user, err = b.db.GetUserByEmail(identifier)
			identifierType = "email"
			LogInfo("Moderator %s attempting purge for email: %s", i.Member.User.Username, identifier)
			break
		}
	}

	if err != nil {
		if err == sql.ErrNoRows {
			LogDebug("Purge failed - user not found with %s: %s", identifierType, identifier)
			b.respondEphemeral(s, i, fmt.Sprintf("❌ No user found with %s: `%s`", identifierType, identifier))
		} else {
			LogError("Error getting user for purge %s: %v", identifier, err)
			b.respondEphemeral(s, i, "❌ Error retrieving user information.")
		}
		return
	}

	// Store user info for logging and notification before deletion
	discordID := user.DiscordID
	discordUsername := user.DiscordUsername
	email := user.Email
	teamRole := user.TeamRole
	isVerified := user.Verified

	// Remove roles if verified
	if isVerified {
		// Remove team role
		if roleID, exists := b.config.Teams[teamRole]; exists {
			if err := s.GuildMemberRoleRemove(b.config.Discord.GuildID, discordID, roleID); err != nil {
				LogWarn("Error removing team role from %s during purge: %v", discordUsername, err)
			} else {
				LogDebug("Removed %s team role from %s", teamRole, discordUsername)
			}
		}

		// Remove members role (if configured)
		if b.config.Discord.MembersRole != "" {
			if err := s.GuildMemberRoleRemove(b.config.Discord.GuildID, discordID, b.config.Discord.MembersRole); err != nil {
				LogWarn("Error removing members role from %s during purge: %v", discordUsername, err)
			} else {
				LogDebug("Removed members role from %s", discordUsername)
			}
		}
	}

	// Delete user from database (GDPR compliance - complete data removal)
	err = b.db.DeleteUser(discordID)
	if err != nil {
		LogError("Error purging user %s from database: %v", discordUsername, err)
		b.respondEphemeral(s, i, "❌ Error deleting user data.")
		return
	}

	LogSuccess("User purged: %s (Email: %s) by moderator %s", discordUsername, email, i.Member.User.Username)
	b.respondEphemeral(s, i, fmt.Sprintf("✅ User data purged successfully.\n\n**User:** %s\n**Email:** %s\n**Discord ID:** %s\n\nAll user data has been permanently removed from the database.", discordUsername, email, discordID))

	// Notify the user
	b.SendDM(discordID, "Your data has been permanently deleted from our system as per GDPR compliance. If you wish to rejoin in the future, you will need to complete the verification process again.")
}

func (b *Bot) handleHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	isAdmin := b.isAdmin(i.Member)

	embed := &discordgo.MessageEmbed{
		Title:       "🛡️ Heimdall Bot Help",
		Description: "Heimdall is your server's gatekeeper, managing user verification through work email authentication.",
		Color:       0x667eea,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "🆕 New Users",
				Value: "When you join the server, I'll send you a DM. Simply reply with your work email address, and I'll send you a verification link. Click the link, select your team, and you're all set!",
			},
			{
				Name:  "📧 Email Requirements",
				Value: "Your email must be from an approved company domain. Each email can only be used once.",
			},
		},
	}

	if isAdmin {
		embed.Fields = append(embed.Fields, []*discordgo.MessageEmbedField{
			{
				Name:  "🔧 Moderator Commands",
				Value: "`/heimdall-stats` - View verification statistics\n`/heimdall-list` - List all users\n`/heimdall-reset` - Reset a user's verification\n`/heimdall-verify` - Manually verify a user\n`/heimdall-changeteam` - Change a user's team\n`/heimdall-restrict` - Temporarily restrict a user's access\n`/heimdall-unrestrict` - Remove restrictions from a user\n`/heimdall-purge` - Permanently delete user data (GDPR)\n`/heimdall-domains` - View approved domains",
			},
		}...)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) isAdmin(member *discordgo.Member) bool {
	if b.config.Discord.AdminRole == "" {
		return false
	}

	for _, roleID := range member.Roles {
		if roleID == b.config.Discord.AdminRole {
			return true
		}
	}

	// Check for Administrator permission
	perms, err := b.session.State.UserChannelPermissions(member.User.ID, member.GuildID)
	if err == nil && (perms&discordgo.PermissionAdministrator != 0) {
		return true
	}

	return false
}

func (b *Bot) respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) AssignRole(userID, roleID string) error {
	return b.session.GuildMemberRoleAdd(b.config.Discord.GuildID, userID, roleID)
}

func (b *Bot) SendDM(userID, message string) error {
	channel, err := b.session.UserChannelCreate(userID)
	if err != nil {
		return err
	}

	_, err = b.session.ChannelMessageSend(channel.ID, message)
	return err
}

func (b *Bot) isApprovedDomain(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	domain := strings.ToLower(parts[1])
	for _, approvedDomain := range b.config.ApprovedDomains {
		if strings.ToLower(approvedDomain) == domain {
			return true
		}
	}

	return false
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func generateVerificationCode() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (b *Bot) getTeamNames() string {
	teams := make([]string, 0, len(b.config.Teams))
	for team := range b.config.Teams {
		teams = append(teams, fmt.Sprintf("`%s`", team))
	}
	return strings.Join(teams, ", ")
}
