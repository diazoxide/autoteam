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

## CLI Commands Enhancement
- Added `--docker-compose-args` flag to `autoteam up` command for passing additional arguments to docker compose
- Supports alias `--args` for shorter usage
- Example: `autoteam up --docker-compose-args="--build --force-recreate"`
- Arguments are automatically parsed and appended to the default `docker compose up -d --remove-orphans` command

## RunOptions Simplification
- **SIMPLIFIED**: Removed OutputFormat, Verbose, and DryRun fields from agent.RunOptions struct
- Claude agent now uses fixed `--output-format stream-json --print` arguments by default
- Qwen agent simplified to only use custom args from configuration
- Monitor.Config no longer includes DryRun field
- Removed `--dry-run` CLI flag from entrypoint command
- All tests pass with simplified agent interface

## Task Output Capture Implementation
- **IMPLEMENTED**: Agent output capture through file-based approach instead of complex RunWithOutput method
- **SIMPLIFIED**: Changed from complex JSON format to simple text list (one task per line)
- Modified FirstLayerPrompt to instruct agents to write simple task list to `/tmp/tasks.txt`
- Updated collectTasksWithAggregationAgent to read tasks from file after agent execution
- Added robust error handling: returns empty task list if file doesn't exist or parsing fails
- Includes file cleanup after successful parsing to prevent stale data
- **NEW**: Simple text parser converts lines to Task objects with generic type and medium priority
- **ENHANCED**: Added comprehensive logging of raw file content for debugging purposes
- Logs content length and full raw content on both success and parse failure scenarios
- All tests pass with simplified file-based task collection workflow

## Simplified Task Prompts
- **SIMPLIFIED**: Completely redesigned prompts to use simple text format instead of complex JSON
- FirstLayerPrompt now asks for simple bullet-point list with human-readable descriptions
- Examples: "In GitHub you have 1 pending PR review for repository owner/repo-name in PR #123"
- SecondLayerPromptTemplate simplified to take just task description as parameter
- **ADDED**: Repository cloning instructions in execution prompt for code access
- Removed complex Task struct fields from prompt building - now uses direct string descriptions
- Much more reliable than JSON parsing - no serialization/deserialization complexity

## Universal Map Merging System
- **FIXED**: settings.service.environment not merging properly in generated compose.yaml
- Implemented universal map merging for all map-type service configurations (not just environment)
- Supports merging for labels, annotations, and any other map fields in Docker Compose services
- Handles multiple map type combinations: `map[string]string`, `map[string]interface{}`, and mixed types
- Agent-specific maps properly override global settings while preserving non-conflicting keys
- Environment variables from global settings now correctly appear in generated compose.yaml

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
- **NEW**: Unified binary directory architecture with comprehensive dependency management
  - Consolidated all binaries (entrypoints, MCP servers, tools) into single `/opt/autoteam/bin` directory
  - Replaced separate `entrypoints` and `bin` directories with unified read/write `/opt/autoteam/bin`
  - Updated Docker Compose volume mounting: `./bin:/opt/autoteam/bin` (read/write, no `:ro` restriction)
  - Enhanced dependency installer with comprehensive existence checking before installation
  - Added smart package management supporting apt, apk, and yum with missing package detection
  - Implemented efficient logging showing which dependencies are already installed vs newly installed
  - Fixed GitHub MCP server installation path conflicts and permission issues
  - Updated MCP server token configuration to use agent-specific tokens (e.g., `${SENIOR_DEVELOPER_GITHUB_TOKEN}`)
  - **Backward Compatibility**: Generator automatically falls back to `/opt/autoteam/entrypoints` if `/opt/autoteam/bin` doesn't exist
  - Migration path: Move existing binaries from `/opt/autoteam/entrypoints` to `/opt/autoteam/bin` when convenient
  - All tests updated and passing with unified architecture
- **NEW**: Implemented streaming logs for executor agent with task-specific log files
  - Each task execution creates separate log file in `logs/{timestamp}-{normalized_task_name}.log` format
  - Log files use timestamp prefix (YYYYMMDD-HHMMSS) and lowercase normalized task names
  - Streaming logs capture agent stdout/stderr immediately after execution (not just at completion)
  - Enhanced `NormalizeTaskText()` function for safe filename generation from notification text
  - Added `StreamingLogger` service in `internal/task/log_stream.go` for log file management
  - Maintains backward compatibility with existing `output.txt` functionality
  - Comprehensive error handling with graceful fallback if log creation fails
- **NEW**: Clean Two-Layer Agent Architecture with Subdirectory Structure
  - Refactored agent naming from suffix-based (`_collector`, `_executor`) to subdirectory-based (`collector/`, `executor/`)
  - Added `GetNormalizedNameWithVariation()` method for clean agent name construction with forward slash preservation
  - Updated entrypoint to pass base agent name and variation separately instead of concatenated strings
  - Fixed agent implementations (Claude, Gemini, Qwen) to use passed agent names instead of re-normalizing internally
  - MCP configuration files now created in proper subdirectories: 
    - `/opt/autoteam/agents/senior_developer/collector/.gemini/settings.json`
    - `/opt/autoteam/agents/senior_developer/executor/mcp.json`
  - Volume mapping maps full agent directory providing access to both collector and executor subdirectories
  - All agent working directories properly set to their respective subdirectories for tool auto-discovery

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

## Simplified Dependency Checking
- All package management and installation logic removed - replaced with simple availability checks
- Claude and Qwen agents now only check if their respective commands exist via `CheckAvailability()` method
- Renamed `Agent.Install()` method to `Agent.CheckAvailability()` for better semantic clarity
- System dependency installer only checks for git and GitHub CLI (gh) availability
- Clear error messages with installation instructions when dependencies are missing
- Users must manually install required dependencies before running AutoTeam
- Removes all complex package manager abstractions and installation logic

## Model Context Protocol (MCP) Server Support
- **NEW**: Comprehensive MCP server integration with auto-team's standard configuration pattern
- Global, agent settings, and agent-level MCP servers with 3-level merging (agent-level > agent.settings > global settings)
- MCP servers configured in autoteam.yaml under `settings.mcp_servers` section
- Environment variable serialization: MCP servers passed to containers via MCP_SERVERS JSON environment variable
- **IMPROVED**: Dedicated MCP configuration files at `/opt/autoteam/agents/{normalized_agent_name}/mcp.json`
- MCP configuration uses correct Claude format with `mcpServers` wrapper object
- Claude execution uses `--mcp-config` parameter to load agent-specific MCP configuration
- No modification of user's personal `~/.claude.json` file - keeps auto-team MCP config isolated
- Uses Agent.GetNormalizedName() for consistent agent name normalization in file paths
- Configuration-only approach: AutoTeam only configures MCP servers in dedicated files, Claude Code handles installation/execution
- Interface-based configuration: Agent.Configurable interface for extensible agent configuration
- Always-run configuration: MCP server configuration runs independently of dependency installation (INSTALL_DEPS setting)
- All tests pass with MCP server integration - no functionality changes to core workflows
- **NEW**: Updated agent interface Run method signature to return (*AgentOutput, error) instead of just error
- Agent output capture: All agent implementations (claude_code.go, qwen_code.go, gemini_cli.go) now capture stdout/stderr in AgentOutput struct
- Enhanced error handling: Agent output is preserved even on execution failures for better debugging and error reporting
- Monitor integration: Updated monitor/loop.go to handle new agent interface signature with proper output handling
- All builds and tests pass with new agent output capture architecture

## Build and Template Workflow
- Use `make build` to build the main autoteam binary (required after template changes due to go:embed)
- Use `make build-entrypoint` to build the entrypoint binary for current platform
- Use `make build-all` to build both main and entrypoint binaries for all supported platforms
- Use `make build-linux` to build both main and entrypoint binaries for Linux platforms (Docker focus)
- Use `make build-entrypoint-all` to build only entrypoint binaries for Linux platforms
- After modifying templates in `internal/generator/templates/`, always rebuild the main binary to update embedded templates
- Use `autoteam generate` to generate compose.yaml and entrypoint.sh from autoteam.yaml configuration

## Parallel Flow Execution System
- **NEW**: True parallel execution for flow steps based on dependency levels
- Level-based dependency resolution groups steps by execution depth (Level 0: no dependencies, Level 1: depends on Level 0, etc.)
- Steps within the same dependency level execute concurrently using goroutines and sync.WaitGroup
- Single-level scenarios: All independent steps execute in parallel simultaneously
- Multi-level scenarios: Each level executes in parallel, waits for completion before proceeding to next level
- Thread-safe step output sharing with proper mutex synchronization for cross-step data access
- Optimized execution: Single steps execute directly without goroutine overhead for efficiency
- Comprehensive error handling: Collects errors from parallel executions and fails fast while preserving partial results
- **PERFORMANCE**: Significantly improves execution time for flows with independent parallel steps
- **BACKWARD COMPATIBLE**: Existing flow configurations work without changes
- All dependency validation and cycle detection preserved from original implementation

## Custom Layer Prompts Configuration (Legacy - Removed)
- **REMOVED**: Two-layer architecture completely replaced with dynamic flow system
- Custom prompts now configured per flow step in `flow.steps[].prompt` field
- Environment variable approach replaced with YAML configuration files per agent
- Simplified configuration: Single `CONFIG_FILE` parameter instead of multiple environment variables

## Configuration Normalization System
- **NEW**: Implemented placeholder variable replacement in environment variables during compose.yaml generation
- **Consistent Syntax**: Single `${AUTOTEAM_VARIABLE_NAME}` format for all placeholders
- **Supported Variables**: 
  - `${AUTOTEAM_AGENT_NAME}` → actual agent name (e.g., "Senior Developer")
  - `${AUTOTEAM_AGENT_DIR}` → agent directory path (e.g., "/opt/autoteam/agents/senior_developer")
  - `${AUTOTEAM_AGENT_NORMALIZED_NAME}` → normalized agent name (e.g., "senior_developer")
- **Implementation**: Added `normalizeEnvironmentValue()` function in generator with runtime value replacement
- **Environment Variable Standardization**: Updated all AutoTeam variables to use consistent `AUTOTEAM_` prefix
- **Example Usage**: In `autoteam.yaml`: `TODO_DB_PATH: ${AUTOTEAM_AGENT_DIR}/todo.db` → In `compose.yaml`: `TODO_DB_PATH: /opt/autoteam/agents/senior_developer/todo.db`
- **Benefits**: Clean configuration management with compile-time variable resolution and consistent naming conventions

## Conditional Flow Execution System
- **NEW**: Added `skip_when` field to FlowStep for conditional step execution
- **Template-Based Conditions**: Uses Go template syntax with Sprig functions for flexible condition evaluation
- **Example**: `skip_when: "{{- index .inputs 0 | trim | eq \"0\" -}}"` skips step if first input is "0"
- **Graceful Error Handling**: Template execution failures log warnings but don't fail the step (assumes should not skip)
- **Input Context**: Skip conditions have access to step inputs from previous step outputs via `.inputs` array
- **Enhanced Logging**: Added detailed step logging showing inputs when starting and outputs when finishing
- **Integration**: Seamlessly works with existing dependency system and parallel execution

## Container Directory Structure
- Codebase is mounted at `/opt/autoteam/codebase` (standard application directory)
- Claude configuration files remain in user home directory: `/home/{user}/.claude` and `/home/{user}/.claude.json`
- Custom volumes can be mounted anywhere as specified in agent settings
