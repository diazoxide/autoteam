#!/bin/bash

echo "Repository: $GITHUB_REPO\n\n"

echo "$PROMPT\n\n"

echo "Installing dependencies...\n\n"
apt update
apt install -y gh jq
npm install -g @anthropic-ai/claude-code -y

echo "Cloning repository...\n\n"
gh repo clone ${GITHUB_REPO} . || git pull

# Ensure the GitHub CLI is authenticated
if ! gh auth status &>/dev/null; then
  echo "Error: GitHub CLI is not authenticated. Please run 'gh auth login' to authenticate."
  exit 1
fi

# Function to check issues and PRs
check_issues_and_prs() {
  # Fetch your GitHub username
  TOTAL=$((
    $(gh pr list --repo "$repo" --search "review-requested:@me is:open" --json number | jq '.[].number' | wc -l) +
    $(gh pr list --repo "$repo" --search "assignee:@me is:open" --json number | jq '.[].number' | wc -l) +
    $(gh issue list --repo "$repo" --search "assignee:@me is:open -linked:pr" --json number | jq '.[].number' | wc -l)
  ))

  echo "$(date): Waiting for your interaction: $TOTAL"

  # Execute command if TOTAL > 0
  if [ $TOTAL -gt 0 ]; then
    echo "$(date): You have $TOTAL open issues or review threads that need your attention."

    echo "$(date): Executing command for $TOTAL pending items..."

    claude --dangerously-skip-permissions --verbose --output-format stream-json --print "$PROMPT\n\n$COMMON_PROMPT" | jq
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
