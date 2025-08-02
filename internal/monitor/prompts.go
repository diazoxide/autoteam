package monitor

import (
	"fmt"
	"time"

	"autoteam/internal/github"
)

// buildReviewRequestPrompt creates a prompt for PR review request notifications
func (m *Monitor) buildReviewRequestPrompt(notification *github.NotificationInfo) string {
	return fmt.Sprintf(`🔍 **PR Review Request Notification**

**Notification Details:**
- **Repository**: %s
- **Subject**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Process this PR review request

**STEP 1 - Validate Actuality**:
First, check if this review request is still valid:
- Use: gh pr view %d --repo %s
- Check if PR is still open
- Verify you're still requested as reviewer
- If PR is merged/closed or review already submitted → Mark as read and explain

**STEP 2 - Perform Review (if actual)**:
If the review request is still valid:
1. **Examine the PR thoroughly**: gh pr view %d --repo %s and gh pr diff %d --repo %s
2. **Review the code changes**: Check quality, logic, security, performance, tests, documentation
3. **Submit your review**: Use gh pr review %d --repo %s with --approve, --request-changes, or --comment

**Review Quality Guidelines**:
- Be constructive and specific in feedback
- Highlight both positive aspects and areas for improvement  
- Reference specific lines or files when possible
- Consider architectural impact and maintainability
- Ensure changes align with project goals

**Expected Output**: Professional code review with clear, actionable feedback.`,
		notification.Repository,
		notification.Subject,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository)
}

// buildAssignedIssuePrompt creates a prompt for assigned issue notifications with intent recognition
func (m *Monitor) buildAssignedIssuePrompt(notification *github.NotificationInfo) string {
	return fmt.Sprintf(`📋 **Assigned Issue Notification**

**Notification Details:**
- **Repository**: %s
- **Subject**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Process this assigned issue intelligently

**STEP 1 - Validate Actuality**:
- Use: gh issue view %d --repo %s
- Verify issue is still open and you're still assigned
- If issue is closed or assignment removed → Mark as read and explain

**STEP 2 - Intent Recognition (CRITICAL)**:
Determine if this is a QUESTION or IMPLEMENTATION request:

**🤔 QUESTION INDICATORS**: "What do you think", "How should we", "Your opinion", "Any thoughts"
**🔨 IMPLEMENTATION INDICATORS**: "Implement", "Fix", "Add", "Create", "Build", "Develop"

**STEP 3A - For QUESTIONS (Consultation)**:
- Read issue thoroughly: gh issue view %d --repo %s --comments
- Provide thoughtful analysis and recommendations
- Comment with your response: gh issue comment %d --repo %s --body "Based on the current architecture, I recommend..."
- **DO NOT CREATE PRs** - this is consultation only

**STEP 3B - For IMPLEMENTATION (Action Required)**:
- Understand requirements: gh issue view %d --repo %s --comments
- Implement the solution: Write code, create necessary files
- Create a pull request: gh pr create --repo %s --title "Fix: [description]" --body "Fixes #%d"
- Link back to issue in your PR

**Intent Examples**:
- "What are your thoughts on adding OAuth?" → QUESTION (comment only)
- "Implement OAuth authentication" → IMPLEMENTATION (create PR)

**Expected Output**: Appropriate response based on detected intent.`,
		notification.Repository,
		notification.Subject,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Repository, notification.Number)
}

// buildAssignedPRPrompt creates a prompt for assigned PR notifications
func (m *Monitor) buildAssignedPRPrompt(notification *github.NotificationInfo) string {
	return fmt.Sprintf(`🔀 **Assigned Pull Request Notification**

**Notification Details:**
- **Repository**: %s
- **Subject**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Process this assigned pull request

**STEP 1 - Validate Actuality**:
- Use: gh pr view %d --repo %s
- Verify PR is still open and you're still assigned
- If PR is merged/closed or assignment removed → Mark as read and explain

**STEP 2 - Handle Assignment (if actual)**:
1. **Review current state**: gh pr view %d --repo %s --comments and gh pr diff %d --repo %s
2. **Determine action needed**: Review, resolve conflicts, make updates, or approve/merge
3. **Take appropriate action**: Use gh pr review, gh pr checkout, or gh pr comment as needed

**Expected Output**: Professional handling of the assigned PR with clear action taken.`,
		notification.Repository,
		notification.Subject,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository)
}

// buildMentionPrompt creates a prompt for mention notifications with consultation vs implementation detection
func (m *Monitor) buildMentionPrompt(notification *github.NotificationInfo) string {
	return fmt.Sprintf(`👤 **Mention Notification**

**Notification Details:**
- **Repository**: %s
- **Subject**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Process this mention intelligently

**STEP 1 - Validate Actuality**:
- Check context: gh issue view %d --repo %s --comments OR gh pr view %d --repo %s --comments
- Verify the mention is still relevant and hasn't been resolved
- If stale/resolved → Mark as read and explain

**STEP 2 - Context Analysis (CRITICAL)**:
Analyze the mention to determine intent:

**🤔 CONSULTATION MENTIONS** (Answer/Advise Only):
- "What do you think @username?"
- "@username, how would you approach this?"
- "@username, any thoughts on..."

**🔨 IMPLEMENTATION MENTIONS** (Action Required):
- "@username, can you implement..."
- "@username, please fix..."
- "@username, help with..."

**STEP 3A - For CONSULTATION**:
- Read full context and provide thoughtful response
- Comment appropriately: gh issue comment %d --repo %s --body "[your analysis]"
- **NO IMPLEMENTATION** - consultation only

**STEP 3B - For IMPLEMENTATION**:
- Understand requirements and take appropriate action
- Create deliverables: Make commits, create PRs as needed
- Follow up with progress and link to your work

**Expected Output**: Appropriate response based on mention context.`,
		notification.Repository,
		notification.Subject,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository)
}

// buildFailedWorkflowPrompt creates a prompt for CI failure notifications
func (m *Monitor) buildFailedWorkflowPrompt(notification *github.NotificationInfo) string {
	return fmt.Sprintf(`⚠️ **CI/Workflow Failure Notification**

**Notification Details:**
- **Repository**: %s
- **Subject**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Fix the failed CI/workflow

**STEP 1 - Validate Actuality**:
- Check current status: gh run list --repo %s --limit 5
- Verify this isn't already fixed by recent commits
- If already fixed → Mark as read and explain

**STEP 2 - Investigate Failure (if actual)**:
1. **Get failure details**: gh run view --repo %s [run-id] --log-failed
2. **Analyze the failure**: Identify specific errors, root cause, related changes
3. **Examine context**: Check which commit triggered the failure

**STEP 3 - Fix the Issue**:
Based on failure type:
- **Build/Compilation**: Fix syntax errors, dependencies, imports
- **Test Failures**: Fix broken tests or update expectations
- **Linting/Formatting**: Run local tools and fix issues
- **Dependencies**: Update packages, resolve conflicts

**STEP 4 - Create Fix**:
1. Make necessary changes and test locally
2. Commit and push: git add . && git commit -m "fix: resolve CI failure" && git push
3. Verify fix: Check that new workflow run passes

**Expected Output**: Working CI/workflow with all issues resolved.`,
		notification.Repository,
		notification.Subject,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.Repository,
		notification.Repository)
}

// buildUnreadCommentPrompt creates a prompt for unread comment notifications
func (m *Monitor) buildUnreadCommentPrompt(notification *github.NotificationInfo) string {
	return fmt.Sprintf(`💬 **Unread Comment Notification**

**Notification Details:**
- **Repository**: %s
- **Subject**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Respond to this comment appropriately

**STEP 1 - Validate Actuality**:
- Check context: gh issue view %d --repo %s --comments OR gh pr view %d --repo %s --comments
- Read latest comments to see if already addressed
- If already resolved → Mark as read and explain

**STEP 2 - Analyze Comment (if actual)**:
1. **Read full context**: Understand discussion thread and history
2. **Identify comment type**: Question, Request, Feedback, or Discussion

**STEP 3 - Respond Appropriately**:
- **Questions**: Provide clear, helpful answers with examples
- **Change Requests**: Acknowledge, make changes, create commits/PRs if needed
- **Code Review**: Address technical feedback and improve code
- **Discussion**: Contribute meaningfully to conversation

**Response Guidelines**:
- Be professional and constructive
- Address all points raised
- Ask for clarification if unclear
- Thank the commenter for their input

**Expected Output**: Thoughtful response that moves the conversation forward.`,
		notification.Repository,
		notification.Subject,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository)
}

// buildGenericNotificationPrompt creates a fallback prompt for generic or unknown notification types
func (m *Monitor) buildGenericNotificationPrompt(notification *github.NotificationInfo) string {
	return fmt.Sprintf(`📢 **Generic GitHub Notification**

**Notification Details:**
- **Reason**: %s
- **Subject**: %s
- **Repository**: %s
- **URL**: %s
- **Updated**: %s
- **Thread ID**: %s

**YOUR MISSION**: Process this GitHub notification intelligently

**STEP 1 - Validate Actuality**:
- Get context: Try gh issue view %d --repo %s OR gh pr view %d --repo %s OR gh repo view %s
- Check if notification context is still relevant
- Verify if any action is actually needed
- If stale/not actionable → Mark as read and explain why

**STEP 2 - Analyze Context (if actual)**:
1. **Understand the subject**: What is this notification about?
2. **Check notification reason**: Why did you receive this?
3. **Review recent activity**: What has happened recently?

**STEP 3 - Take Appropriate Action**:
- **If informational only**: Acknowledge if needed, no further action
- **If requires attention**: Read content, take appropriate action
- **If unclear**: Ask for clarification before acting

**Default Actions to Consider**:
- Check if you need to respond to someone
- See if there are pending tasks for you
- Determine if any follow-up is needed
- Provide helpful input if relevant

**Expected Output**: Context-appropriate response based on specific notification content.`,
		notification.Reason,
		notification.Subject,
		notification.Repository,
		notification.URL,
		notification.UpdatedAt.Format(time.RFC3339),
		notification.ThreadID,
		notification.Number, notification.Repository,
		notification.Number, notification.Repository,
		notification.Repository)
}
