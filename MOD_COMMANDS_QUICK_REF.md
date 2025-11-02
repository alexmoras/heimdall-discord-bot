# Moderator Commands - Quick Reference

## 🚀 Quick Commands

### Manually Verify a User
```
/heimdall-verify @user email:user@company.com team:Engineering
```
Instantly verifies a user without email flow.

### Change User's Team
```
/heimdall-changeteam @user team:Product
```
Moves user from current team to new team.

### Temporarily Restrict User
```
/heimdall-unverify @user reason:"Subscription payment overdue"
```
Removes roles but keeps data for quick reactivation.

### Reactivate Restricted User
```
/heimdall-reverify @user
```
Instantly restores access using saved data.

## 📋 Command Comparison

| Command | Purpose | Requirements | Changes Roles | Sends Email | Keeps Data |
|---------|---------|--------------|---------------|-------------|------------|
| `/heimdall-verify` | Manual verification | User + email + team | ✅ Assigns both | ❌ No | ✅ Yes |
| `/heimdall-changeteam` | Change team | User (must be verified) + team | ✅ Swaps team role | ❌ No | ✅ Yes |
| `/heimdall-unverify` | Temporary restriction | User (must be verified) + reason | ❌ Removes all | ❌ No | ✅ Yes |
| `/heimdall-reverify` | Reactivate | User (must be unverified) | ✅ Restores all | ❌ No | ✅ Yes |
| `/heimdall-reset` | Remove verification | User | ❌ Removes all | ❌ No | ❌ No |

## 🎯 Common Scenarios

### New Hire (No Email Yet)
```
/heimdall-verify @NewHire email:newhire@company.com team:Engineering
```

### Transfer Between Teams
```
/heimdall-changeteam @Employee team:Product
```

### Subscription Lapse
```
# Payment overdue
/heimdall-unverify @Member reason:"Payment overdue"

# Payment received
/heimdall-reverify @Member
```

### Temporary Suspension
```
# Suspend for 7 days
/heimdall-unverify @User reason:"Suspended until 2025-11-08"

# Suspension ends
/heimdall-reverify @User
```

### Fix Wrong Team Assignment
```
/heimdall-changeteam @Employee team:Engineering
```

### Start Over (Reset)
```
/heimdall-reset @Employee
# Then they can verify normally or use /heimdall-verify
```

## ⚠️ Important Notes

- ✅ Both commands require **admin/moderator permissions**
- ✅ Email must be from **approved domain** (for verify)
- ✅ Team must exist in **config.yaml**
- ✅ User receives **DM notification**
- ✅ **Members role** assigned automatically (if configured)
- ❌ Cannot verify already-verified user (use changeteam instead)
- ❌ Cannot change team of unverified user (use verify first)

## 🔍 Error Messages

| Error | Meaning | Solution |
|-------|---------|----------|
| "Invalid email format" | Email not valid | Check email spelling |
| "Domain not approved" | Email domain not in list | Use approved domain |
| "Email already registered" | Email in use | Use different email |
| "User already verified" | User exists | Use `/heimdall-changeteam` |
| "Team not found" | Team not in config | Check team name spelling |
| "User not verified" | User not in system | Use `/heimdall-verify` first |

## 🔄 Workflow Examples

### Onboard 5 New Interns
```
/heimdall-verify @Intern1 email:intern1@company.com team:Engineering
/heimdall-verify @Intern2 email:intern2@company.com team:Engineering
/heimdall-verify @Intern3 email:intern3@company.com team:Product
/heimdall-verify @Intern4 email:intern4@company.com team:Design
/heimdall-verify @Intern5 email:intern5@company.com team:Marketing
```

### Reorganize Team
```
# Move 3 engineers to new product team
/heimdall-changeteam @Engineer1 team:Product
/heimdall-changeteam @Engineer2 team:Product
/heimdall-changeteam @Engineer3 team:Product
```

### Fix Verification Issue
```
# User verified but no role assigned
/heimdall-reset @User
/heimdall-verify @User email:user@company.com team:Engineering
```

## 📊 Verification Status Check

View all users and their status:
```
/heimdall-list
```

View statistics:
```
/heimdall-stats
```

## 💡 Best Practices

### ✅ DO
- Use for special cases (VIPs, issues, bulk onboarding)
- Verify email belongs to the user
- Double-check team spelling
- Use for users who can't receive email

### ❌ DON'T
- Use as default (email flow is better)
- Bypass domain restrictions
- Verify without confirming user identity
- Change teams without user knowledge

## 🆘 Need Help?

Full documentation: [MODERATOR_COMMANDS.md](MODERATOR_COMMANDS.md)

---

**Quick Test:**
```
# Test verify command
/heimdall-verify @TestUser email:test@approvedomain.com team:Engineering

# Test change team
/heimdall-changeteam @TestUser team:Product

# Test list
/heimdall-list
```

**Version:** 1.0.0