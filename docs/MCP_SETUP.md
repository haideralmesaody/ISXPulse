# MCP (Model Context Protocol) Setup Guide

This guide explains how to configure and use GitHub and Sentry MCP servers with Claude Code for the ISX Pulse project.

## Overview

MCP servers enhance Claude Code's capabilities by providing direct access to external services. We've configured two MCP servers:

1. **GitHub MCP** - Repository management, PR reviews, issue tracking
2. **Sentry MCP** - Error monitoring, performance tracking, incident management

## Prerequisites

### 1. Install NPM Dependencies

The MCP servers run via npx, so ensure Node.js and npm are installed:

```bash
node --version  # Should be 18.0.0 or higher
npm --version   # Should be 8.0.0 or higher
```

### 2. Create Authentication Tokens

#### GitHub Personal Access Token

1. Go to https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Name: "ISX Pulse MCP Integration"
4. Select scopes:
   - `repo` (Full control of private repositories)
   - `workflow` (Update GitHub Action workflows)
   - `read:org` (Read organization membership)
   - `gist` (Create gists)
5. Generate and copy the token

#### Sentry Auth Token

1. Go to https://sentry.io/settings/account/api/auth-tokens/
2. Click "Create New Token"
3. Name: "ISX Pulse MCP Integration"
4. Select scopes:
   - `project:read`
   - `project:write`
   - `event:read`
   - `org:read`
5. Create and copy the token

## Configuration

### 1. Set Environment Variables

Create a `.env` file in the project root (copy from `.env.example`):

```bash
cp .env.example .env
```

Edit `.env` and add your tokens:

```env
# GitHub MCP
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx

# Sentry MCP
SENTRY_AUTH_TOKEN=sntrys_xxxxxxxxxxxxxxxxxxxx
SENTRY_ORG=your-org-name
SENTRY_PROJECT=isx-pulse
```

### 2. Verify MCP Configuration

The following files have been configured:

- `.mcp.json` - Defines MCP server configurations
- `.claude/settings.local.json` - Enables MCP servers for the project

## Usage

### GitHub MCP Commands

Once configured, you can use these commands in Claude Code:

```
# Review a pull request
Review PR #456 and suggest improvements

# Check recent issues
What are the open issues with the 'bug' label?

# Create an issue
Create a GitHub issue for the license activation bug we found

# Check workflow status
What's the status of our CI/CD workflows?
```

### Sentry MCP Commands

```
# Check recent errors
What are the most common errors in the last 24 hours?

# Analyze performance
Show me the slowest transactions in the past week

# Review specific error
Analyze error event ID xyz123 and suggest a fix

# Check release health
How is the latest release performing?
```

## Authentication

When first using an MCP server, Claude Code will prompt for authentication:

1. Use the `/mcp` command in Claude Code
2. Select the server to authenticate (github or sentry)
3. Follow the OAuth flow if prompted
4. Tokens are securely stored for future sessions

## Security Best Practices

1. **Never commit tokens** - The `.env` file is gitignored
2. **Use minimal scopes** - Only grant necessary permissions
3. **Rotate tokens regularly** - Regenerate tokens every 90 days
4. **Monitor usage** - Check token usage in GitHub/Sentry settings
5. **Revoke if compromised** - Immediately revoke and regenerate if exposed

## Troubleshooting

### MCP Server Not Available

If Claude Code can't find the MCP servers:

1. Restart Claude Code session
2. Verify `.mcp.json` exists in project root
3. Check that `enableAllProjectMcpServers` is `true` in settings

### Authentication Failed

If authentication fails:

1. Verify tokens are correct in `.env`
2. Check token hasn't expired
3. Ensure token has required scopes
4. Try `/mcp` command to re-authenticate

### Server Connection Issues

If MCP servers can't connect:

```bash
# Test GitHub token
curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user

# Test Sentry token
curl -H "Authorization: Bearer $SENTRY_AUTH_TOKEN" https://sentry.io/api/0/
```

## Integration with ISX Pulse

### GitHub Integration Benefits

- Automated PR reviews for code quality
- Issue tracking for bugs and features
- Workflow monitoring for CI/CD
- Commit history analysis

### Sentry Integration Benefits

- Real-time error monitoring
- Performance bottleneck identification
- User impact analysis
- Release tracking and rollback decisions

## Advanced Usage

### Custom MCP Workflows

Create custom workflows combining both MCPs:

```
# Error-to-Issue workflow
1. Identify critical errors in Sentry
2. Create GitHub issues automatically
3. Link errors to PRs for fixes
4. Track resolution through both platforms
```

### Monitoring Dashboard

Use MCP data for a unified view:

```
# Combined metrics
- GitHub: Open PRs, issue velocity, commit frequency
- Sentry: Error rate, performance metrics, user sessions
- ISX Pulse: License activations, operation success rate
```

## Future Enhancements

Potential additional MCP servers for ISX Pulse:

1. **PostgreSQL MCP** - Direct database queries
2. **Google Sheets MCP** - Native sheets integration
3. **Custom ISX MCP** - Direct ISX API access
4. **Grafana MCP** - Metrics visualization

## Support

For MCP-related issues:

1. Check this documentation
2. Review Claude Code MCP docs: https://docs.anthropic.com/en/docs/claude-code/mcp
3. Check server-specific docs:
   - GitHub: https://github.com/modelcontextprotocol/server-github
   - Sentry: https://github.com/modelcontextprotocol/server-sentry