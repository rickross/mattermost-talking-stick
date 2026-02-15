# Mattermost Talking Stick

Dynamic channel moderation and speaking privilege management for live events, panels, and Q&A sessions.

## Features

- **Channel Modes**: Open, Speakers Only, Q&A, and Locked
- **Speaking Privileges**: Grant/revoke dynamically via slash commands
- **Q&A Sessions**: Managed question slots for audience participation
- **Configurable Bypass**: System/Team/Channel admins and bots can bypass restrictions
- **Real-time Control**: No server restart needed

## Installation

1. Download the latest release `.tar.gz`
2. Upload via System Console → Plugins → Plugin Management
3. Enable the plugin
4. Configure bypass settings if desired

## Usage

### Grant/Revoke Speaking Privileges

```
/stick grant @username          # Grant speaking privileges
/stick revoke @username         # Revoke speaking privileges
/stick list                     # List current speakers and status
```

### Channel Modes

```
/stick mode open                # Everyone can post (default)
/stick mode speakers            # Only granted speakers can post
/stick mode qa                  # Speakers + Q&A participants can post
/stick mode locked              # Only administrators can post
```

### Q&A Mode

```
/stick qa-grant @username       # Grant 1 question slot
/stick qa-grant @username 3     # Grant 3 question slots
```

## Use Cases

- **Live Events**: Manage speakers during webinars or conferences
- **Panel Discussions**: Control who's "on stage"
- **Q&A Sessions**: Let audience members ask questions in order
- **AI Agent Coordination**: Manage which agents can speak
- **Town Halls**: CEO announcements with controlled audience participation

## Configuration

Configure which roles can bypass talking stick restrictions:

- **Allow System Admins to Bypass** (default: true)
- **Allow Team Admins to Bypass** (default: true)
- **Allow Channel Admins to Bypass** (default: true)
- **Allow Bots to Bypass** (default: true)

## Building from Source

```bash
make deps    # Download dependencies
make build   # Build for all platforms
make dist    # Create plugin package
```

## License

Apache 2.0

## Author

The GIT School - Where consciousness studies itself, together.
