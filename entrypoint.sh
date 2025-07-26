#!/bin/bash

if [ "$INSTALL_DEPS" == "true" ]; then
  echo "Installing dependencies...\n\n"
  apt update
  apt install -y gh jq
  npm install -g @anthropic-ai/claude-code -y
fi

# region Git

# Ensure the GitHub CLI is authenticated
if ! gh auth status &>/dev/null; then
  echo "Error: GitHub CLI is not authenticated. GH_TOKEN must be set."
  exit 1
fi


echo -e "Repository: $GITHUB_REPO\n"
echo -e "AGENT_NAME: $AGENT_NAME\n\n"

GH_USER=$(gh api graphql -f query='query { viewer { login } }' --jq '.data.viewer.login')
echo -e "User: $GH_USER\n"

# === Set up credential helper to use stored credentials ===
git config --global credential.helper store

# === Create credentials file with HTTPS token ===
echo "https://${GH_USER}:${GH_TOKEN}@github.com/${GITHUB_REPO}" > ~/.git-credentials

# === Optional: Set user identity for commits ===
git config --global user.name "$GH_USER"
git config --global user.email "$GH_USER@users.noreply.github.com"


cd ~/codebase
echo -e "Cloning repository...\n\n"
gh repo clone ${GITHUB_REPO} . || echo "already cloned"

GITHUB_MAIN_BRANCH=$(gh repo view --json defaultBranchRef --jq .defaultBranchRef.name)
echo -e "Default branch: $GITHUB_MAIN_BRANCH\n\n"

gh_my_pending_list() {
  OWNER=$(cut -d/ -f1 <<< "$GITHUB_REPO")
  NAME=$(cut -d/ -f2 <<< "$GITHUB_REPO")

  local response
  response=$(gh api graphql -f query='
  {
    requestedPRs: search(query: "repo:'"$OWNER"'/'"$NAME"' is:pr is:open review-requested:@me", type: ISSUE, first: 50) {
      nodes {
        ... on PullRequest {
          number
          title
          url
        }
      }
    }
    assignedPRs: search(query: "repo:'"$OWNER"'/'"$NAME"' is:pr is:open assignee:@me", type: ISSUE, first: 50) {
      nodes {
        ... on PullRequest {
          number
          title
          url
        }
      }
    }
    assignedIssues: search(query: "repo:'"$OWNER"'/'"$NAME"' is:issue is:open assignee:@me -linked:pr", type: ISSUE, first: 50) {
      nodes {
        ... on Issue {
          number
          title
          url
        }
      }
    }
    myOpenPRs: search(query: "repo:'"$OWNER"'/'"$NAME"' is:pr is:open author:@me", type: ISSUE, first: 50) {
      nodes {
        ... on PullRequest {
          number
          title
          url
          reviews(last: 20) {
            nodes {
              author {
                login
              }
              state
              submittedAt
            }
          }
        }
      }
    }
  }')

  # 📥 Review Requests
  local reviews=$(jq -r '.data.requestedPRs.nodes | select(length > 0) | map("- [#\(.number)](\(.url)) \(.title)") | .[]?' <<< "$response")
  if [[ -n "$reviews" ]]; then
    echo "📥 Review Requests:"
    echo "$reviews"
    echo ""
  fi

  # 🧷 Assigned PRs
  local assigned_prs=$(jq -r '.data.assignedPRs.nodes | select(length > 0) | map("- [#\(.number)](\(.url)) \(.title)") | .[]?' <<< "$response")
  if [[ -n "$assigned_prs" ]]; then
    echo "🧷 Assigned PRs:"
    echo "$assigned_prs"
    echo ""
  fi

  # 🚧 Assigned Issues
  local issues=$(jq -r '.data.assignedIssues.nodes | select(length > 0) | map("- [#\(.number)](\(.url)) \(.title)") | .[]?' <<< "$response")
  if [[ -n "$issues" ]]; then
    echo "🚧 Assigned Issues (no PR):"
    echo "$issues"
    echo ""
  fi

  # 🛠 My PRs with Changes Requested
  local changes=$(jq -r '
    .data.myOpenPRs.nodes
    | map(select(
        (.reviews.nodes
          | group_by(.author.login)
          | map(.[-1])
          | map(select(.state == "CHANGES_REQUESTED"))
          | length) > 0))
    | select(length > 0)
    | map("- [#\(.number)](\(.url)) \(.title)")
    | .[]?' <<< "$response")

  if [[ -n "$changes" ]]; then
    echo "🛠 My PRs with Changes Requested:"
    echo "$changes"
    echo ""
  fi
}

# Function to check issues and PRs
check_issues_and_prs() {
  PENDING_LIST=$(gh_my_pending_list)

  echo "$(date): Checking for pending items..."

  # Execute command if PENDING_LIST is not empty
  if [ -n "$PENDING_LIST" ]; then
    echo "$(date): You have pending items that need your attention:"
    echo "$PENDING_LIST"

    # Switch always to main branch and pull latest changes
    echo -e "$(date): Switching to main branch...\n"
    git fetch
    git checkout ${GITHUB_MAIN_BRANCH}
    git reset --hard origin/${GITHUB_MAIN_BRANCH}
    echo -e "$(date): Git pull completed.\n"

    IMPORTANT_PROMPT="IMPORTANT: Submit only one Pull Request per iteration. Avoid the '1 PR = 1 commit' approach. Large Pull Requests should be broken down into multiple small, logical commits that each represent a cohesive change."
    LARGE_PROMPT="$PENDING_LIST\n\n$IMPORTANT_PROMPT\n\n$AGENT_PROMPT\n\n$COMMON_PROMPT"

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

    echo -e "\n\n$(date): Pull request submission completed.\n"
    return 1
  fi

  return 0
}

# Run initial check
check_issues_and_prs

# Start periodic refresh every 10 minutes (600 seconds)
while true; do
  sleep 60
  echo "$(date): Refreshing counts..."

  check_issues_and_prs
done
