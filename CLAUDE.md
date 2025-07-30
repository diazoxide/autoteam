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
- Successfully implemented dotenv support for both `autoteam` and `entrypoint` commands using godotenv
- Added Docker Compose stack naming using team_name from config via `-p` flag
- Implemented urfave/cli Before hook pattern for global config loading and context passing
- All tests pass - context-based architecture working correctly
- Docker Compose commands now use configured team_name instead of default "autoteam"

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
