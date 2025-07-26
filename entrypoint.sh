#!/bin/bash
echo -e "Repository: $GITHUB_REPO\n"
echo -e "AGENT_NAME: $AGENT_NAME\n\n"

if [ "$INSTALL_DEPS" == "true" ]; then
  echo "Installing dependencies...\n\n"
  apt update
  apt install -y gh jq
  npm install -g @anthropic-ai/claude-code -y
fi

cd ~/codebase
echo -e "Cloning repository...\n\n"
gh repo clone ${GITHUB_REPO} . || echo "already cloned"

# Ensure the GitHub CLI is authenticated
if ! gh auth status &>/dev/null; then
  echo "Error: GitHub CLI is not authenticated. Please run 'gh auth login' to authenticate."
  exit 1
fi

# Function to check issues and PRs
check_issues_and_prs() {
  # Fetch your GitHub username
  TOTAL=$((
    $(gh pr list --repo "$GITHUB_REPO" --search "review-requested:@me is:open" --json number | jq '.[].number' | wc -l) +
    $(gh pr list --repo "$GITHUB_REPO" --search "assignee:@me is:open" --json number | jq '.[].number' | wc -l) +
    $(gh issue list --repo "$GITHUB_REPO" --search "assignee:@me is:open -linked:pr" --json number | jq '.[].number' | wc -l)
  ))

  echo "$(date): Waiting for your interaction: $TOTAL"

  # Execute command if TOTAL > 0
  if [ $TOTAL -gt 0 ]; then
    GITHUB_MAIN_BRANCH=$(gh repo view --json defaultBranchRef --jq .defaultBranchRef.name)
    echo "Default branch: $GITHUB_MAIN_BRANCH\n\n"

    # Switch always to main branch and pull latest changes
    git checkout ${GITHUB_MAIN_BRANCH}
    git fetch
    git reset --hard origin/${GITHUB_MAIN_BRANCH}

    echo "$(date): You have $TOTAL open issues or review threads that need your attention."

    IMPORTANT_PROMPT="IMPORTANT: Submit only one Pull Request per iteration. Avoid the '1 PR = 1 commit' approach. Large Pull Requests should be broken down into multiple small, logical commits that each represent a cohesive change."
    LARGE_PROMPT="$IMPORTANT_PROMPT\n\n$AGENT_PROMPT\n\n$COMMON_PROMPT"

    AGENT_REPO_PROMPT="./.autoteam/agent-${AGENT_NAME}.md"
    COMMON_REPO_PROMPT="./.autoteam/common.md"

    if [ -f "$COMMON_REPO_PROMPT" ]; then
      LARGE_PROMPT="$LARGE_PROMPT\n\n$(cat $COMMON_REPO_PROMPT)"
    fi
    if [ -f "$AGENT_REPO_PROMPT" ]; then
      LARGE_PROMPT="$LARGE_PROMPT\n\n$(cat $AGENT_REPO_PROMPT)"
    fi

    echo -e "$LARGE_PROMPT\n\n"

    claude --dangerously-skip-permissions --verbose --output-format stream-json --print "$LARGE_PROMPT" | jq
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
