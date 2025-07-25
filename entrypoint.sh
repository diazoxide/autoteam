#!/bin/bash

apt update

apt install -y gh jq

npm install -g @anthropic-ai/claude-code -y

gh repo clone ${GITHUB_REPO} . || git pull


# Ensure the GitHub CLI is authenticated
if ! gh auth status &>/dev/null; then
  echo "Error: GitHub CLI is not authenticated. Please run 'gh auth login' to authenticate."
  exit 1
fi

# Set the repository
GITHUB_REPO="diazoxide/godlejump"

# Function to check issues and PRs
check_issues_and_prs() {
  # Fetch your GitHub username
  USER=$(gh api graphql -f query='query { viewer { login } }' --jq '.data.viewer.login')

  # Count open issues assigned to you without an associated pull request
  ISSUE_COUNT=$(gh issue list --repo "$GITHUB_REPO" --search "assignee:$USER is:open is:issue -linked:pr" --json number --jq 'length')

  # Count unresolved review threads you've commented on
  PR_COUNT=$(gh api graphql -f owner="${GITHUB_REPO%/*}" -f repo="${GITHUB_REPO#*/}" -f query='
    query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        pullRequests(states: OPEN, first: 50) {
          nodes {
            reviewThreads(first: 50) {
              nodes {
                isResolved
                comments(first: 20) {
                  nodes {
                    author { login }
                  }
                }
              }
            }
          }
        }
      }
    }
  ' | jq --arg USERNAME "$USER" '
    [
      .data.repository.pullRequests.nodes
      | .[]
      | .reviewThreads.nodes
      | .[]
      | select(.isResolved == false)
      | select(any(.comments.nodes[]; .author.login == $USERNAME))
    ]
    | length
  ')

  # Calculate total
  TOTAL=$((ISSUE_COUNT + PR_COUNT))

  # Output the results
  echo "$(date): Open issues assigned to you without a PR: $ISSUE_COUNT"
  echo "$(date): Unresolved review threads you've commented on: $PR_COUNT"
  echo "$(date): Total: $TOTAL"

  # Execute command if TOTAL > 0
  if [ $TOTAL -gt 0 ]; then
    echo "$(date): You have $TOTAL open issues or review threads that need your attention."
    # TODO: Replace this with the actual command you want to run
    echo "$(date): Executing command for $TOTAL pending items..."

    claude --dangerously-skip-permissions --verbose --output-format stream-json --print "$(cat ~/prompt.md)" | jq
  fi

  return $TOTAL
}

# Run initial check
check_issues_and_prs

# Start periodic refresh every 10 minutes (600 seconds)
while true; do
  sleep 60
  echo "$(date): Refreshing counts..."
  check_issues_and_prs
done
