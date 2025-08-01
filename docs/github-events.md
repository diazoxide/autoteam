# GitHub Events Tracked by AutoTeam

AutoTeam monitors various GitHub events to ensure agents respond to all important activities. This document describes all the event types that AutoTeam tracks and how they are prioritized.

## Event Types

### 1. Review Requests (Priority: 1000)
- **Description**: Pull requests where the agent is requested as a reviewer
- **Action**: Agent reviews the PR and provides feedback
- **Example**: When someone requests your review on their PR

### 2. Mentions (Priority: 900)
- **Description**: Comments where the agent is @mentioned in issues or PRs
- **Action**: Agent responds to the mention or takes requested action
- **Example**: "@agent can you help with this bug?"

### 3. Assigned PRs (Priority: 800)
- **Description**: Pull requests assigned to the agent
- **Action**: Agent works on implementing or fixing the PR
- **Example**: A PR assigned to the agent for implementation

### 4. Failed Workflows (Priority: 700)
- **Description**: GitHub Actions workflows that failed on the agent's PRs
- **Action**: Agent investigates and fixes the failing tests/checks
- **Example**: CI tests failing on a PR authored by the agent

### 5. PRs with Changes Requested (Priority: 600)
- **Description**: Agent's pull requests that have changes requested by reviewers
- **Action**: Agent addresses the feedback and re-requests review
- **Example**: Reviewer requests changes on the agent's PR

### 6. Unread Comments (Priority: 500)
- **Description**: New comments on issues/PRs the agent is participating in
- **Action**: Agent reads and responds if necessary
- **Example**: Someone comments on an issue the agent is working on

### 7. Assigned Issues (Priority: 400)
- **Description**: Issues assigned to the agent (excluding those with linked PRs)
- **Action**: Agent works on resolving the issue
- **Example**: A bug report or feature request assigned to the agent

### 8. Notifications (Priority: 300)
- **Description**: Unread GitHub notifications from watched repositories
- **Action**: Agent reviews and acts on important notifications
- **Example**: New releases, security alerts, or other repository events

## Priority System

AutoTeam uses a sophisticated priority system to determine which items to work on first:

### Base Priority Scores
Each event type has a base priority score (shown above). Higher scores mean higher priority.

### Priority Modifiers

1. **Age Bonus**: Older items get priority boosts
   - 7+ days old: +300 points
   - 3+ days old: +200 points
   - 1+ day old: +100 points

2. **Urgency Keywords**: Items with urgent keywords get +500 points
   - Keywords: urgent, critical, blocker, hotfix, emergency, p0, sev1

3. **High Priority Keywords**: Items with these keywords get +100 points
   - Keywords: bug, fix, error, broken, failing, p1, sev2

4. **Failure Penalty**: Items that have failed before get -100 points per failure
   - Helps prevent agents from getting stuck on impossible tasks

## Implementation Details

### Mentions and Comments
- AutoTeam searches for mentions in the last 7 days
- Only mentions with "@username" format are detected
- Comments are fetched for issues/PRs where the agent is participating

### Notifications
- Uses GitHub's Notifications API
- Only fetches unread notifications
- Filtered by repository patterns configured in AutoTeam

### Failed Workflows
- Monitors workflow runs from the last 24 hours
- Only includes workflows with "failure" conclusion
- Associates workflows with related pull requests

### Repository Filtering
All events respect the repository include/exclude patterns configured in `autoteam.yaml`. This ensures agents only work on repositories they're assigned to.

## Configuration

No additional configuration is needed for these event types. They are automatically tracked when AutoTeam runs with proper GitHub authentication.

### Required Permissions
The GitHub token used by agents must have these permissions:
- `repo` - Full repository access
- `notifications` - Read notifications
- `workflow` - Read workflow runs

## Cooldown and Retry Logic

- Failed items enter a cooldown period before being retried
- Cooldown duration increases with each failure
- Items are retried up to the configured maximum attempts (default: 3)
- This prevents agents from repeatedly failing on the same item