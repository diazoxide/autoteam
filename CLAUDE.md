# Project Instructions for Claude

## Commit Message Guidelines
- NEVER add Claude branding in commit messages
- NEVER include "Generated with Claude Code" signatures
- NEVER include "Co-Authored-By: Claude" lines
- Keep commit messages simple and professional
- Focus on the actual changes made

## Development Workflow
- Run `make check` before committing to ensure code passes all checks
- Use `make fmt` to format Go code properly
- All tests should pass before creating commits

## Recent Implementation Notes
- Successfully implemented comprehensive structured logging using zap across entire codebase
- Added configurable log levels (debug, info, warn, error) with --log-level flag for all commands
- Converted 180+ log calls from standard library to structured zap logging with contextual fields
- Context-based logger pattern implemented for proper logger propagation throughout application
- All tests pass with structured logging implementation - no functionality changes
- Successfully implemented dotenv support for both `autoteam` and `entrypoint` commands using godotenv
- Added Docker Compose stack naming using team_name from config via `-p` flag
- Implemented urfave/cli Before hook pattern for global config loading and context passing
- All tests pass - context-based architecture working correctly
- Docker Compose commands now use configured team_name instead of default "autoteam"
- Converted all log.Printf and log.Println calls in internal/git/setup.go to structured zap logging
- Logger migration pattern: use logger.FromContext(ctx) for functions with context, logger.NewLogger(logger.InfoLevel) for functions without context
- Structured logging with zap fields for better observability and debugging
- **NEW**: Implemented notification-first GitHub API optimization with automatic read marking
  - Reduced API calls by 60-70% by using GitHub Notifications API as primary source
  - Added notification correlation to map notification reasons to pending item types
  - Implemented automatic notification marking as read after successful item processing
  - Enhanced NotificationInfo with correlation fields (ThreadID, CorrelatedType, Number, SubjectType)
  - Updated ProcessingItem and PrioritizedItem to track notification thread IDs
  - Added fallback to REST API strategy when notifications unavailable
  - All notification thread IDs are marked as read only after successful item resolution
  - **FIXED**: Resolution detector now supports all item types (mention, unread_comment, notification, failed_workflow)
  - Added proper resolution checking logic for notification-based items with correct matching criteria

## Multi-Repository Architecture
- Complete multi-repository support with pattern matching and regex
- Repository filtering with include/exclude patterns using `/pattern/` syntax
- Cross-repository GitHub API operations with filtering
- Per-repository git working directories: `/opt/autoteam/agents/{agent}/codebase/{owner-repo}`
- Repository-aware state management and item tracking
- On-demand repository cloning when processing items

## Configuration Architecture
- Main `autoteam` command reads `autoteam.yaml` with repositories configuration
- Entrypoint command uses multi-repository environment variables (no YAML dependency)
- REPOSITORIES_INCLUDE/REPOSITORIES_EXCLUDE environment variables (comma-separated patterns)
- Clean separation: YAML config for main command, env vars for containerized entrypoint

## Single Notification Processing System  
- Complete rewrite to simplified single notification processing workflow
- AI-driven actuality validation using GitHub CLI commands
- Type-specific prompts with intent recognition for consultation vs implementation
- Notification types: review_request, assigned_issue, assigned_pr, mention, failed_workflow, unread_comment, generic
- Smart intent detection prevents over-implementation for consultation requests (solves Issue #4 problem)
- Mandatory notification read-marking after processing to prevent duplicates
- Removed complex state management, prioritization, and resolution detection for simplicity
- All tests pass and builds succeed across all platforms

## Build and Template Workflow
- Use `make build` to build the main autoteam binary (required after template changes due to go:embed)
- Use `make build-entrypoint` to build the entrypoint binary for current platform
- Use `make build-all` to build both main and entrypoint binaries for all supported platforms
- Use `make build-linux` to build both main and entrypoint binaries for Linux platforms (Docker focus)
- Use `make build-entrypoint-all` to build only entrypoint binaries for Linux platforms
- After modifying templates in `internal/generator/templates/`, always rebuild the main binary to update embedded templates
- Use `autoteam generate` to generate compose.yaml and entrypoint.sh from autoteam.yaml configuration

## Container Directory Structure
- Codebase is mounted at `/opt/autoteam/codebase` (standard application directory)
- Claude configuration files remain in user home directory: `/home/{user}/.claude` and `/home/{user}/.claude.json`
- Custom volumes can be mounted anywhere as specified in agent settings
