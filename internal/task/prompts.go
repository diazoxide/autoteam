package task

import (
	"fmt"
)

// FirstLayerPrompt is used by the aggregation agent to collect tasks from MCP servers
const FirstLayerPrompt = `TASK: Query MCP servers and create task list

Your job is to find actionable tasks using available tools and write them to /tmp/tasks.txt

Use your available tools to check for:
- Pending items that need attention
- Tasks requiring action
- Notifications or alerts

Write results to /tmp/tasks.txt with one task per line. Create the file even if no tasks are found.

Examples:
- GitHub: 1 pending PR review for repo/name in PR #123
- GitHub: 1 assigned issue #456 in repo/name  
- Platform: 1 unread notification requiring response

Create /tmp/tasks.txt now.`

// SecondLayerPromptTemplate is used by the execution agent to handle a specific task
const SecondLayerPromptTemplate = `You are a task execution agent. Execute this specific task thoroughly and professionally.

## Task Description
%s

## Your Mission
Complete this task using available MCP servers and tools.

## Important Instructions
1. **Repository Access**: If you need to access a repository's code:
   - First check if the repository exists in the codebase directory
   - If the repository doesn't exist, clone it first using git commands
   - Then navigate to the repository directory to work with the code

2. **Execute Thoroughly**: Take all necessary actions to complete the task
3. **Use MCP Servers**: Leverage available MCP servers for platform-specific operations (GitHub, Slack, etc.)
4. **Be Professional**: Maintain high quality standards in all interactions
5. **Provide Summary**: Briefly summarize what you accomplished

## Common Task Types
- **PR Review**: Examine code changes, provide constructive feedback, approve/request changes
- **Issue Implementation**: Understand requirements, implement solution, create PR
- **Mention Response**: Read context, provide helpful response or take requested action
- **Message Response**: Respond appropriately based on message content and urgency
- **Failed Workflow**: Investigate failure, fix issues, ensure CI passes
- **Comment Response**: Read full thread, respond to questions or address feedback

Begin task execution now.`

// BuildSecondLayerPrompt creates a prompt for the execution agent
func BuildSecondLayerPrompt(taskDescription string) string {
	return fmt.Sprintf(SecondLayerPromptTemplate, taskDescription)
}
