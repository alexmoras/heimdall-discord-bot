package main

import (
	"fmt"
	"net/smtp"
)

type EmailService struct {
	config *Config
}

func NewEmailService(config *Config) *EmailService {
	return &EmailService{config: config}
}

func (e *EmailService) SendVerificationEmail(toEmail, verificationCode, username string) error {
	verificationURL := fmt.Sprintf("%s/verify?code=%s", e.config.Server.BaseURL, verificationCode)

	subject := "Verify Your Discord Account"
	
	// Plain text version
	plainBody := fmt.Sprintf(`Hello %s,

Welcome to the server! Please verify your email address by clicking the link below:

%s

This link will allow you to:
1. Confirm your email address
2. Select your team role

Once you've completed these steps, you'll be granted access to the server.

If you didn't request this verification, please ignore this email.

Best regards,
The Heimdall Bot Team`, username, verificationURL)

	// HTML version with clickable link
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .button { display: inline-block; padding: 15px 30px; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white !important; text-decoration: none; border-radius: 5px; font-weight: bold; margin: 20px 0; }
        .button:hover { opacity: 0.9; }
        .steps { background: white; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .footer { text-align: center; color: #666; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🛡️ Heimdall Verification</h1>
        </div>
        <div class="content">
            <p>Hello <strong>%s</strong>,</p>
            <p>Welcome to the server! Please verify your email address to complete your registration.</p>
            
            <center>
                <a href="%s" class="button">Verify My Email</a>
            </center>
            
            <p style="text-align: center; color: #666; font-size: 14px;">Or copy and paste this link into your browser:<br>
            <code style="background: #e0e0e0; padding: 5px 10px; border-radius: 3px; word-break: break-all;">%s</code></p>
            
            <div class="steps">
                <p><strong>This link will allow you to:</strong></p>
                <ol>
                    <li>Confirm your email address</li>
                    <li>Select your team role</li>
                </ol>
                <p>Once you've completed these steps, you'll be granted access to the server.</p>
            </div>
            
            <p style="color: #666; font-size: 14px;">If you didn't request this verification, please ignore this email.</p>
        </div>
        <div class="footer">
            <p>Best regards,<br>The Heimdall Bot Team</p>
        </div>
    </div>
</body>
</html>`, username, verificationURL, verificationURL)

	// Create multipart message with both plain text and HTML
	message := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n"+
		"\r\n"+
		"--boundary123\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n"+
		"--boundary123\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n"+
		"--boundary123--\r\n",
		e.config.Email.FromName, e.config.Email.FromAddress,
		toEmail,
		subject,
		plainBody,
		htmlBody)

	auth := smtp.PlainAuth("",
		e.config.Email.SMTPUsername,
		e.config.Email.SMTPPassword,
		e.config.Email.SMTPHost,
	)

	addr := fmt.Sprintf("%s:%d", e.config.Email.SMTPHost, e.config.Email.SMTPPort)

	err := smtp.SendMail(addr, auth, e.config.Email.FromAddress, []string{toEmail}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}