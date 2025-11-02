package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type WebServer struct {
	config    *Config
	db        *Database
	bot       *Bot
	startTime time.Time
}

func NewWebServer(config *Config, db *Database, bot *Bot) *WebServer {
	return &WebServer{
		config:    config,
		db:        db,
		bot:       bot,
		startTime: time.Now(),
	}
}

// truncateCode safely truncates a verification code for logging
func truncateCode(code string) string {
	if len(code) <= 8 {
		return code
	}
	return code[:8] + "..."
}

func (ws *WebServer) Start() error {
	http.HandleFunc("/verify", ws.handleVerify)
	http.HandleFunc("/api/verify", ws.handleAPIVerify)
	http.HandleFunc("/health", ws.handleHealth)
	http.HandleFunc("/status", ws.handleStatus)

	addr := fmt.Sprintf(":%d", ws.config.Server.Port)
	return http.ListenAndServe(addr, nil)
}

func (ws *WebServer) handleVerify(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Verification code is required", http.StatusBadRequest)
		return
	}

	LogDebug("Web verification page accessed with code: %s", truncateCode(code))

	user, err := ws.db.GetUserByVerificationCode(code)
	if err != nil {
		LogWarn("Invalid verification code accessed: %s", truncateCode(code))
		http.Error(w, "Invalid or expired verification code", http.StatusNotFound)
		return
	}

	if user.Verified {
		LogDebug("Already verified user accessing verification page: %s", user.DiscordUsername)
		ws.renderAlreadyVerified(w, user)
		return
	}

	LogDebug("Rendering verification page for: %s", user.DiscordUsername)
	ws.renderVerificationPage(w, user)
}

func (ws *WebServer) handleAPIVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code string `json:"code"`
		Team string `json:"team"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Code == "" || req.Team == "" {
		http.Error(w, "Code and team are required", http.StatusBadRequest)
		return
	}

	LogDebug("Web verification attempt: code=%s team=%s", truncateCode(req.Code), req.Team)

	user, err := ws.db.GetUserByVerificationCode(req.Code)
	if err != nil {
		LogWarn("Invalid verification code used: %s", truncateCode(req.Code))
		http.Error(w, "Invalid verification code", http.StatusNotFound)
		return
	}

	if user.Verified {
		LogDebug("User %s already verified, ignoring web verification", user.DiscordUsername)
		http.Error(w, "User already verified", http.StatusBadRequest)
		return
	}

	LogInfo("Processing web verification for %s (team: %s)", user.DiscordUsername, req.Team)

	// Check if team exists
	roleID, exists := ws.config.Teams[req.Team]
	if !exists {
		LogWarn("Invalid team selected: %s (user: %s)", req.Team, user.DiscordUsername)
		http.Error(w, "Invalid team selection", http.StatusBadRequest)
		return
	}

	// Update database
	if err := ws.db.UpdateUserTeam(user.DiscordID, req.Team); err != nil {
		LogError("Error updating user team for %s: %v", user.DiscordUsername, err)
		http.Error(w, "Failed to verify user", http.StatusInternalServerError)
		return
	}

	// Assign base members role (if configured)
	if ws.config.Discord.MembersRole != "" {
		if err := ws.bot.AssignRole(user.DiscordID, ws.config.Discord.MembersRole); err != nil {
			LogError("Error assigning members role to %s: %v", user.DiscordUsername, err)
			// Don't fail the request, but log the error
		} else {
			LogDebug("Assigned members role to %s", user.DiscordUsername)
		}
	}

	// Assign team role in Discord
	if err := ws.bot.AssignRole(user.DiscordID, roleID); err != nil {
		LogError("Error assigning team role to %s: %v", user.DiscordUsername, err)
		// Don't fail the request, but log the error
	} else {
		LogDebug("Assigned %s team role to %s", req.Team, user.DiscordUsername)
	}

	// Send success DM
	ws.bot.SendDM(user.DiscordID, fmt.Sprintf("✅ Verification complete! Welcome to the %s team. You now have access to the server.", req.Team))

	LogSuccess("User %s verified successfully (team: %s, email: %s)", user.DiscordUsername, req.Team, user.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Verification complete!",
	})
}

func (ws *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (ws *WebServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Get database statistics
	total, verified, pending, err := ws.db.GetStats()
	if err != nil {
		LogError("Error getting stats for status endpoint: %v", err)
		http.Error(w, "Failed to retrieve statistics", http.StatusInternalServerError)
		return
	}

	// Check bot connection status
	botConnected := ws.bot.session != nil && ws.bot.session.DataReady

	// Calculate uptime
	uptime := time.Since(ws.startTime)

	// Build status response
	status := map[string]interface{}{
		"status":  "healthy",
		"version": Version,
		"uptime":  uptime.String(),
		"uptime_seconds": int(uptime.Seconds()),
		"bot": map[string]interface{}{
			"connected": botConnected,
		},
		"database": map[string]interface{}{
			"total_users":    total,
			"verified_users": verified,
			"pending_users":  pending,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

func (ws *WebServer) renderVerificationPage(w http.ResponseWriter, user *User) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Heimdall - Verify Your Account</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 500px;
            width: 100%;
            padding: 40px;
        }
        .logo {
            text-align: center;
            font-size: 48px;
            margin-bottom: 10px;
        }
        h1 {
            text-align: center;
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }
        .subtitle {
            text-align: center;
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .user-info {
            background: #f7f7f7;
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 25px;
        }
        .user-info p {
            margin: 5px 0;
            color: #555;
            font-size: 14px;
        }
        .user-info strong {
            color: #333;
        }
        label {
            display: block;
            margin-bottom: 8px;
            color: #333;
            font-weight: 600;
        }
        select {
            width: 100%;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 16px;
            margin-bottom: 20px;
            transition: border-color 0.3s;
        }
        select:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 20px rgba(102, 126, 234, 0.4);
        }
        button:active {
            transform: translateY(0);
        }
        button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }
        .success-message, .error-message {
            padding: 12px;
            border-radius: 8px;
            margin-top: 20px;
            display: none;
            text-align: center;
            font-weight: 500;
        }
        .success-message {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .error-message {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">🛡️</div>
        <h1>Heimdall Verification</h1>
        <p class="subtitle">Complete your verification to access the server</p>
        
        <div class="user-info">
            <p><strong>Discord:</strong> {{.DiscordUsername}}</p>
            <p><strong>Email:</strong> {{.Email}}</p>
        </div>

        <form id="verifyForm">
            <label for="team">Select Your Team:</label>
            <select id="team" name="team" required>
                <option value="">-- Choose a team --</option>
                {{range $team, $roleID := .Teams}}
                <option value="{{$team}}">{{$team}}</option>
                {{end}}
            </select>

            <button type="submit" id="submitBtn">Complete Verification</button>
        </form>

        <div class="success-message" id="successMsg">
            ✅ Verification complete! You can now close this page and return to Discord.
        </div>
        <div class="error-message" id="errorMsg"></div>
    </div>

    <script>
        document.getElementById('verifyForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const team = document.getElementById('team').value;
            const submitBtn = document.getElementById('submitBtn');
            const successMsg = document.getElementById('successMsg');
            const errorMsg = document.getElementById('errorMsg');
            
            if (!team) {
                errorMsg.textContent = 'Please select a team';
                errorMsg.style.display = 'block';
                return;
            }
            
            submitBtn.disabled = true;
            submitBtn.textContent = 'Verifying...';
            errorMsg.style.display = 'none';
            
            try {
                const urlParams = new URLSearchParams(window.location.search);
                const code = urlParams.get('code');
                
                const response = await fetch('/api/verify', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ code, team }),
                });
                
                if (!response.ok) {
                    const data = await response.text();
                    throw new Error(data || 'Verification failed');
                }
                
                successMsg.style.display = 'block';
                document.getElementById('verifyForm').style.display = 'none';
            } catch (error) {
                errorMsg.textContent = error.message;
                errorMsg.style.display = 'block';
                submitBtn.disabled = false;
                submitBtn.textContent = 'Complete Verification';
            }
        });
    </script>
</body>
</html>`

	t, err := template.New("verify").Parse(tmpl)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		DiscordUsername string
		Email           string
		Teams           map[string]string
	}{
		DiscordUsername: user.DiscordUsername,
		Email:           user.Email,
		Teams:           ws.config.Teams,
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}

func (ws *WebServer) renderAlreadyVerified(w http.ResponseWriter, user *User) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Already Verified</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 500px;
            width: 100%;
            padding: 40px;
            text-align: center;
        }
        .logo {
            font-size: 64px;
            margin-bottom: 20px;
        }
        h1 {
            color: #333;
            margin-bottom: 15px;
        }
        p {
            color: #666;
            line-height: 1.6;
        }
        .team-badge {
            display: inline-block;
            margin-top: 20px;
            padding: 10px 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border-radius: 20px;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">✅</div>
        <h1>Already Verified</h1>
        <p>Your account has already been verified.</p>
        <div class="team-badge">{{.TeamRole}}</div>
        <p style="margin-top: 20px;">You can close this page and return to Discord.</p>
    </div>
</body>
</html>`

	t, err := template.New("already_verified").Parse(tmpl)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		TeamRole string
	}{
		TeamRole: user.TeamRole,
	}

	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, data)
}