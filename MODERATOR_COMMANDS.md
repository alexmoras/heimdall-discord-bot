# Moderator Commands

## Overview

Heimdall includes two powerful moderator commands that allow manual user management without requiring users to go through the email verification process.

## Commands

### `/heimdall-verify` - Manual Verification

Manually verify a user by providing their Discord account, email, and team.

**Usage:**
```
/heimdall-verify user:@username email:user@company.com team:Engineering
```

**Parameters:**
- `user` - The Discord user to verify (mention or select)
- `email` - Their work email address (must be from approved domain)
- `team` - The team to assign them to

**What it does:**
1. ✅ Validates email format
2. ✅ Checks email domain is in approved list
3. ✅ Ensures email isn't already registered
4. ✅ Creates user record in database
5. ✅ Assigns Members role (if configured)
6. ✅ Assigns Team role
7. ✅ Sends confirmation DM to user
8. ✅ Marks user as verified

**Use Cases:**
- Onboard users who can't receive email (no access to work email yet)
- Verify VIPs or special guests quickly
- Fix issues where email verification failed
- Bulk onboarding for new team members

**Example:**
```
/heimdall-verify user:@JohnDoe email:john@company.com team:Engineering
```

Result: John is instantly verified, assigned to Engineering team, and gets access.

### `/heimdall-changeteam` - Change User Team

Change a verified user's team, automatically removing the old team role and assigning the new one.

**Usage:**
```
/heimdall-changeteam user:@username team:Product
```

**Parameters:**
- `user` - The Discord user to change (must be already verified)
- `team` - The new team to assign them to

**What it does:**
1. ✅ Verifies user is already verified
2. ✅ Removes old team role
3. ✅ Assigns new team role
4. ✅ Updates database
5. ✅ Sends notification DM to user
6. ✅ Keeps Members role intact

**Use Cases:**
- User switches teams/departments
- Reorganization or team changes
- Fix incorrect team assignment
- Promote or transfer users

**Example:**
```
/heimdall-changeteam user:@JohnDoe team:Product
```

Result: John's Engineering role is removed, Product role is assigned, and database is updated.

### `/heimdall-restrict` - Temporarily Restrict User

Temporarily remove a user's access without deleting their data. Perfect for subscription lapses or temporary suspensions.

**Usage:**
```
/heimdall-restrict user:@username reason:"Unpaid subscription"
```

**Parameters:**
- `user` - The Discord user to restrict (must be verified)
- `reason` - Optional reason for restriction (shown to user in DM)

**What it does:**
1. ✅ Removes team role
2. ✅ Removes members role
3. ✅ Marks user as "restricted" in database
4. ✅ Keeps email and team data saved
5. ✅ Blocks user from DM verification
6. ✅ Sends notification DM to user
7. ✅ Can be quickly reversed with `/heimdall-unrestrict`

**Use Cases:**
- Subscription or membership payment lapses
- Temporary suspensions for rule violations
- Compliance holds while issues are resolved
- Seasonal access (e.g., summer break)

**Example:**
```
/heimdall-restrict user:@JohnDoe reason:"Subscription payment overdue"
```

Result: John loses all roles immediately, receives DM with reason, cannot use bot to re-verify, but data is saved for quick restoration.

### `/heimdall-unrestrict` - Remove Restrictions from User

Quickly restore access for a user who was previously restricted.

**Usage:**
```
/heimdall-unrestrict user:@username
```

**Parameters:**
- `user` - The Discord user to unrestrict (must be in restricted state)

**What it does:**
1. ✅ Restores members role
2. ✅ Restores original team role
3. ✅ Marks user as verified again
4. ✅ Sends welcome back DM
5. ✅ Uses saved email/team data
6. ✅ Instant restoration

**Use Cases:**
- Payment received after lapse
- Suspension period ended
- Compliance requirements met
- Issue resolved

**Example:**
```
/heimdall-unrestrict user:@JohnDoe
```

Result: John's roles are restored instantly using saved data, receives welcome back DM.

### `/heimdall-purge` - Permanently Delete User Data (GDPR)

Permanently delete all user data from the database for GDPR/privacy compliance. Can delete by Discord user or email address.

**Usage:**
```
/heimdall-purge user:@username
/heimdall-purge email:user@company.com
```

**Parameters:**
- `user` - The Discord user to purge (option 1)
- `email` - The email address to purge (option 2)

Provide either `user` OR `email`, not both.

**What it does:**
1. ✅ Removes all Discord roles
2. ✅ Permanently deletes user record from database
3. ✅ Removes all personal data (email, username, etc.)
4. ✅ Logs the deletion action
5. ✅ Sends notification DM to user
6. ❌ Data cannot be recovered after purge

**Use Cases:**
- GDPR data deletion requests
- User requests account removal
- Cleaning up after former members
- Privacy compliance requirements

**Examples:**
```
/heimdall-purge user:@JohnDoe
/heimdall-purge email:john@company.com
```

Result: All of John's data is permanently removed from the database. If they want to rejoin, they must complete verification again from scratch.

## Permissions

Both commands require **admin permissions**. The bot checks:
1. User has the configured `admin_role` in config.yaml, OR
2. User has Discord Administrator permission

## Validation & Safety

### Email Validation (`/heimdall-verify`)
- ✅ Email format must be valid (regex check)
- ✅ Domain must be in `approved_domains` list
- ✅ Email must not already be registered
- ✅ User must not already be verified
- ❌ Blocks duplicate registrations

### Team Validation (Both Commands)
- ✅ Team must exist in `teams` config
- ✅ Shows available teams if invalid team specified
- ❌ Cannot assign non-existent team

### User Validation
- ✅ User must exist in Discord server
- ✅ For `/heimdall-changeteam`: User must be verified first
- ❌ Cannot change team of unverified user

## Workflow Examples

### Example 1: New Hire Onboarding

**Scenario:** New employee joining but won't have email access until first day.

**Solution:**
```
/heimdall-verify user:@NewEmployee email:new.employee@company.com team:Engineering
```

Employee gets instant access on their first day!

### Example 2: Department Transfer

**Scenario:** Engineer moves to Product team.

**Solution:**
```
/heimdall-changeteam user:@Engineer team:Product
```

Roles updated instantly, user notified of change.

### Example 3: Bulk Onboarding

**Scenario:** Onboarding 10 new interns.

**Solution:**
```
/heimdall-verify user:@Intern1 email:intern1@company.com team:Engineering
/heimdall-verify user:@Intern2 email:intern2@company.com team:Product
/heimdall-verify user:@Intern3 email:intern3@company.com team:Design
...
```

All interns verified in minutes!

### Example 4: Fix Verification Issue

**Scenario:** User went through email verification but role wasn't assigned.

**Option 1:** Reset and re-verify
```
/heimdall-reset user:@User
```
Then user goes through email flow again.

**Option 2:** Manual verify (faster)
```
/heimdall-verify user:@User email:user@company.com team:Engineering
```

Issue fixed immediately!

### Example 5: Subscription Lapse

**Scenario:** User's membership payment failed.

**Solution:**
```
/heimdall-restrict user:@User reason:"Subscription payment overdue - please update billing"
```

User loses access but data is preserved.

**When they pay:**
```
/heimdall-unrestrict user:@User
```

Access restored instantly!

### Example 6: Temporary Suspension

**Scenario:** User violated rules, suspended for 7 days.

**Solution:**
```
/heimdall-restrict user:@User reason:"Temporary suspension - ends 2025-11-08"
```

**After 7 days:**
```
/heimdall-unrestrict user:@User
```

User welcomed back automatically!

### Example 7: GDPR Deletion Request

**Scenario:** Former member requests all their data be deleted.

**Solution:**
```
/heimdall-purge user:@FormerMember
```

All data permanently removed, GDPR compliance met!

## Error Messages

### `/heimdall-verify` Errors

**"Invalid email format"**
- Email doesn't match format `user@domain.com`
- Fix: Check email spelling

**"Domain not approved"**
- Email domain not in `approved_domains` list
- Fix: Use approved domain or add domain to config

**"Email already registered"**
- Another user already using this email
- Each email can only be used once
- Fix: Use different email or reset other user

**"User already verified"**
- User is already verified
- Fix: Use `/heimdall-changeteam` to change their team instead

**"Team not found"**
- Team doesn't exist in config
- Shows list of available teams
- Fix: Use exact team name from config or add team to config

### `/heimdall-changeteam` Errors

**"User not verified"**
- User hasn't completed verification yet
- Fix: Use `/heimdall-verify` to verify them first

**"Team not found"**
- Team doesn't exist in config
- Fix: Use exact team name or add team to config

**"Already on this team"**
- User is already assigned to the requested team
- No change needed

## Database Impact

### `/heimdall-verify`
Creates new database record:
```sql
INSERT INTO users (discord_id, discord_username, email, verification_code, team_role, verified)
VALUES (user_id, username, email, code, team, TRUE)
```

### `/heimdall-changeteam`
Updates existing record:
```sql
UPDATE users SET team_role = new_team WHERE discord_id = user_id
```

## Audit Trail

Both commands log actions:

**Verification:**
```
Moderator @Admin manually verified @User (user@company.com) for team Engineering
```

**Team Change:**
```
Moderator @Admin changed @User from Engineering to Product team
```

**User Notifications:**
- User receives DM notification for both actions
- DM includes what happened and which moderator did it (user is told "by a moderator")

## Best Practices

### When to Use Manual Verification

✅ **Good use cases:**
- New hires without email access yet
- VIP guests or partners
- Fixing verification issues quickly
- Bulk onboarding events
- Users with email delivery problems

❌ **Avoid for:**
- Regular user onboarding (use email flow)
- Bypassing domain restrictions (maintain security)
- Users who can receive email normally

### When to Use Change Team

✅ **Good use cases:**
- Department transfers
- Role changes
- Fixing incorrect team assignments
- Team reorganizations

❌ **Avoid for:**
- Unverified users (use `/heimdall-verify`)
- As a workaround for verification issues

### Security Considerations

1. **Limit moderator access** - Only give admin role to trusted users
2. **Verify emails carefully** - Ensure email belongs to the user
3. **Use approved domains** - Don't bypass domain restrictions
4. **Keep records** - Bot logs all actions
5. **Audit regularly** - Review manual verifications periodically

## Comparison with Email Verification

| Feature | Email Verification | Manual Verification |
|---------|-------------------|---------------------|
| **Speed** | ~2-5 minutes | Instant |
| **Email Required** | Yes | Yes (but doesn't send) |
| **User Action** | User completes form | Moderator does everything |
| **Verification** | Email ownership proven | Moderator vouches for user |
| **Best For** | Normal onboarding | Special cases |
| **Security** | High (email ownership) | Medium (trust moderator) |

## Configuration

No additional configuration needed! Commands use existing settings:
- `approved_domains` - Email validation
- `teams` - Team selection
- `admin_role` - Permission checking
- `members_role` - Role assignment

## Testing

### Test Manual Verification
```
1. /heimdall-verify user:@TestUser email:test@approvedomain.com team:Engineering
2. Check user has both Members role and Engineering role
3. Check user received DM notification
4. Verify in database: /heimdall-list
```

### Test Team Change
```
1. /heimdall-changeteam user:@TestUser team:Product
2. Check Engineering role removed, Product role added
3. Check user received DM notification
4. Verify in database: /heimdall-list
```

### Test Error Handling
```
1. Try invalid email: /heimdall-verify user:@User email:notanemail team:Eng
   Expected: "Invalid email format"

2. Try unapproved domain: /heimdall-verify user:@User email:test@bad.com team:Eng
   Expected: "Domain not approved"

3. Try invalid team: /heimdall-verify user:@User email:test@good.com team:FakeTeam
   Expected: "Team 'FakeTeam' not found"

4. Change unverified user: /heimdall-changeteam user:@NewUser team:Product
   Expected: "User not verified"
```

## Summary

**Manual Verification (`/heimdall-verify`):**
- Instantly verify users without email flow
- Requires: user mention, email (approved domain), team
- Perfect for special cases and quick onboarding

**Change Team (`/heimdall-changeteam`):**
- Move users between teams instantly
- Requires: user mention (must be verified), new team
- Perfect for transfers and role changes

**Restrict (`/heimdall-restrict`):**
- Temporarily remove access while preserving data
- Requires: user mention, optional reason
- Perfect for subscription lapses and temporary suspensions

**Unrestrict (`/heimdall-unrestrict`):**
- Quickly restore access for restricted users
- Requires: user mention (must be restricted)
- Instant restoration using saved data

**Purge (`/heimdall-purge`):**
- Permanently delete all user data (GDPR)
- Requires: user mention OR email address
- Complete data removal, cannot be recovered

All commands:
- ✅ Admin/Moderator only
- ✅ Full validation and error checking
- ✅ Database and Discord role sync
- ✅ User notifications via DM
- ✅ Safe and auditable

These tools make Heimdall flexible for both automated and manual user management! 🎯