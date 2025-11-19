package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

type User struct {
	ID              int64
	DiscordID       string
	DiscordUsername string
	Email           string
	VerificationCode string
	TeamRole        string
	Verified        bool
	Unverified      bool // True if user has been unverified by moderator
	CreatedAt       time.Time
	VerifiedAt      *time.Time
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		discord_id TEXT UNIQUE NOT NULL,
		discord_username TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		verification_code TEXT UNIQUE NOT NULL,
		team_role TEXT,
		verified BOOLEAN DEFAULT FALSE,
		unverified BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		verified_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_discord_id ON users(discord_id);
	CREATE INDEX IF NOT EXISTS idx_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_verification_code ON users(verification_code);
	`

	_, err := d.db.Exec(schema)
	return err
}

func (d *Database) CreateUser(discordID, username, email, verificationCode string) error {
	query := `
		INSERT INTO users (discord_id, discord_username, email, verification_code)
		VALUES (?, ?, ?, ?)
	`
	_, err := d.db.Exec(query, discordID, username, email, verificationCode)
	return err
}

func (d *Database) GetUserByDiscordID(discordID string) (*User, error) {
	query := `
		SELECT id, discord_id, discord_username, email, verification_code, 
		       COALESCE(team_role, ''), verified, COALESCE(unverified, 0), created_at, verified_at
		FROM users WHERE discord_id = ?
	`
	
	var user User
	var verifiedAt sql.NullTime
	
	err := d.db.QueryRow(query, discordID).Scan(
		&user.ID, &user.DiscordID, &user.DiscordUsername, &user.Email,
		&user.VerificationCode, &user.TeamRole, &user.Verified, &user.Unverified,
		&user.CreatedAt, &verifiedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	if verifiedAt.Valid {
		user.VerifiedAt = &verifiedAt.Time
	}
	
	return &user, nil
}

func (d *Database) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, discord_id, discord_username, email, verification_code,
		       COALESCE(team_role, ''), verified, COALESCE(unverified, 0), created_at, verified_at
		FROM users WHERE email = ?
	`

	var user User
	var verifiedAt sql.NullTime

	err := d.db.QueryRow(query, email).Scan(
		&user.ID, &user.DiscordID, &user.DiscordUsername, &user.Email,
		&user.VerificationCode, &user.TeamRole, &user.Verified, &user.Unverified,
		&user.CreatedAt, &verifiedAt,
	)

	if err != nil {
		return nil, err
	}

	if verifiedAt.Valid {
		user.VerifiedAt = &verifiedAt.Time
	}

	return &user, nil
}

func (d *Database) GetUserByUsername(username string) (*User, error) {
	query := `
		SELECT id, discord_id, discord_username, email, verification_code,
		       COALESCE(team_role, ''), verified, COALESCE(unverified, 0), created_at, verified_at
		FROM users WHERE discord_username = ?
	`

	var user User
	var verifiedAt sql.NullTime

	err := d.db.QueryRow(query, username).Scan(
		&user.ID, &user.DiscordID, &user.DiscordUsername, &user.Email,
		&user.VerificationCode, &user.TeamRole, &user.Verified, &user.Unverified,
		&user.CreatedAt, &verifiedAt,
	)

	if err != nil {
		return nil, err
	}

	if verifiedAt.Valid {
		user.VerifiedAt = &verifiedAt.Time
	}

	return &user, nil
}

func (d *Database) GetUserByVerificationCode(code string) (*User, error) {
	query := `
		SELECT id, discord_id, discord_username, email, verification_code, 
		       COALESCE(team_role, ''), verified, created_at, verified_at
		FROM users WHERE verification_code = ?
	`
	
	var user User
	var verifiedAt sql.NullTime
	
	err := d.db.QueryRow(query, code).Scan(
		&user.ID, &user.DiscordID, &user.DiscordUsername, &user.Email,
		&user.VerificationCode, &user.TeamRole, &user.Verified,
		&user.CreatedAt, &verifiedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	if verifiedAt.Valid {
		user.VerifiedAt = &verifiedAt.Time
	}
	
	return &user, nil
}

func (d *Database) UpdateUserTeam(discordID, teamRole string) error {
	query := `
		UPDATE users
		SET team_role = ?, verified = TRUE, verified_at = CURRENT_TIMESTAMP
		WHERE discord_id = ?
	`
	_, err := d.db.Exec(query, teamRole, discordID)
	return err
}

func (d *Database) MarkUserVerified(discordID string) error {
	query := `
		UPDATE users
		SET verified = TRUE, verified_at = CURRENT_TIMESTAMP
		WHERE discord_id = ?
	`
	_, err := d.db.Exec(query, discordID)
	return err
}

func (d *Database) UnverifyUser(discordID string) error {
	query := `
		UPDATE users 
		SET verified = FALSE, unverified = TRUE
		WHERE discord_id = ?
	`
	_, err := d.db.Exec(query, discordID)
	return err
}

func (d *Database) ReverifyUser(discordID string) error {
	query := `
		UPDATE users 
		SET verified = TRUE, unverified = FALSE, verified_at = CURRENT_TIMESTAMP
		WHERE discord_id = ?
	`
	_, err := d.db.Exec(query, discordID)
	return err
}

func (d *Database) DeleteUser(discordID string) error {
	query := `DELETE FROM users WHERE discord_id = ?`
	_, err := d.db.Exec(query, discordID)
	return err
}

func (d *Database) GetAllUsers() ([]User, error) {
	query := `
		SELECT id, discord_id, discord_username, email, verification_code, 
		       COALESCE(team_role, ''), verified, COALESCE(unverified, 0), created_at, verified_at
		FROM users ORDER BY created_at DESC
	`
	
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []User
	for rows.Next() {
		var user User
		var verifiedAt sql.NullTime
		
		err := rows.Scan(
			&user.ID, &user.DiscordID, &user.DiscordUsername, &user.Email,
			&user.VerificationCode, &user.TeamRole, &user.Verified, &user.Unverified,
			&user.CreatedAt, &verifiedAt,
		)
		if err != nil {
			return nil, err
		}
		
		if verifiedAt.Valid {
			user.VerifiedAt = &verifiedAt.Time
		}
		
		users = append(users, user)
	}
	
	return users, rows.Err()
}

func (d *Database) GetStats() (total, verified, pending int, err error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN verified = TRUE THEN 1 ELSE 0 END), 0) as verified,
			COALESCE(SUM(CASE WHEN verified = FALSE THEN 1 ELSE 0 END), 0) as pending
		FROM users
	`
	
	err = d.db.QueryRow(query).Scan(&total, &verified, &pending)
	return
}

func (d *Database) Close() error {
	return d.db.Close()
}

// Check if Discord ID exists
func (d *Database) DiscordIDExists(discordID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE discord_id = ?)`
	err := d.db.QueryRow(query, discordID).Scan(&exists)
	return exists, err
}

// Check if email exists
func (d *Database) EmailExists(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`
	err := d.db.QueryRow(query, email).Scan(&exists)
	return exists, err
}