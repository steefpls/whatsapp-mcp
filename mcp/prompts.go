package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerPrompts defines all MCP prompts for common workflows.
func (m *MCPServer) registerPrompts() {
	m.server.AddPrompt(
		mcp.NewPrompt("search_person_messages",
			mcp.WithPromptDescription("Find ALL messages from a specific person across all WhatsApp chats to understand context about them"),
			mcp.WithArgument("contact_name",
				mcp.ArgumentDescription("Name of the person whose messages you want to find"),
				mcp.RequiredArgument(),
			),
		),
		m.handleSearchPersonMessagesPrompt,
	)

	m.server.AddPrompt(
		mcp.NewPrompt("get_context_about_person",
			mcp.WithPromptDescription("Get comprehensive context about someone by analyzing all their messages"),
			mcp.WithArgument("contact_name",
				mcp.ArgumentDescription("Name of the person to analyze"),
				mcp.RequiredArgument(),
			),
			mcp.WithArgument("focus",
				mcp.ArgumentDescription("Focus area: 'recent' for recent activity only, 'all' for complete history (default: all)"),
			),
		),
		m.handleGetContextAboutPersonPrompt,
	)

	m.server.AddPrompt(
		mcp.NewPrompt("analyze_conversation",
			mcp.WithPromptDescription("Analyze recent messages from a specific conversation"),
			mcp.WithArgument("contact_name",
				mcp.ArgumentDescription("Name of the contact or group"),
				mcp.RequiredArgument(),
			),
		),
		m.handleAnalyzeConversationPrompt,
	)

	m.server.AddPrompt(
		mcp.NewPrompt("search_keyword",
			mcp.WithPromptDescription("Search for specific text or keywords across all WhatsApp conversations"),
			mcp.WithArgument("keyword",
				mcp.ArgumentDescription("Text or keyword to search for"),
				mcp.RequiredArgument(),
			),
		),
		m.handleSearchKeywordPrompt,
	)
}

// handleSearchPersonMessagesPrompt handles the search_person_messages prompt request.
func (m *MCPServer) handleSearchPersonMessagesPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	contactName := req.Params.Arguments["contact_name"]
	if contactName == "" {
		contactName = "[contact name]"
	}

	promptText := `Find ALL messages from ` + contactName + ` across every chat.

1. find_chat(search="` + contactName + `") to get their JID
2. search_messages(from="<JID>") — omit query entirely

Group results by chat and date so I can see patterns across contexts.`

	return mcp.NewGetPromptResult(
		"Find all messages from "+contactName,
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewTextContent(promptText),
			),
		},
	), nil
}

// handleGetContextAboutPersonPrompt handles the get_context_about_person prompt request.
func (m *MCPServer) handleGetContextAboutPersonPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	contactName := req.Params.Arguments["contact_name"]
	if contactName == "" {
		contactName = "[contact name]"
	}

	focus := req.Params.Arguments["focus"]
	if focus == "" {
		focus = "all"
	}

	limitHint := ""
	if focus == "recent" {
		limitHint = " with limit=50 for recent only"
	}

	promptText := `Build context about ` + contactName + ` from their WhatsApp messages.

1. find_chat(search="` + contactName + `") to get JID
2. search_messages(from="<JID>")` + limitHint + `

Summarize: communication patterns, main topics, tone, key facts to remember.`

	return mcp.NewGetPromptResult(
		"Get context about "+contactName,
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewTextContent(promptText),
			),
		},
	), nil
}

// handleAnalyzeConversationPrompt handles the analyze_conversation prompt request.
func (m *MCPServer) handleAnalyzeConversationPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	contactName := req.Params.Arguments["contact_name"]
	if contactName == "" {
		contactName = "[contact name]"
	}

	promptText := `Analyze the recent WhatsApp conversation with ` + contactName + `.

1. find_chat(search="` + contactName + `") to get chat_jid
2. get_chat_messages(chat_jid="<jid>", limit=50)

Summarize: main topics, action items, important dates, tone, key takeaways.`

	return mcp.NewGetPromptResult(
		"Analyze conversation with "+contactName,
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewTextContent(promptText),
			),
		},
	), nil
}

// handleSearchKeywordPrompt handles the search_keyword prompt request.
func (m *MCPServer) handleSearchKeywordPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	keyword := req.Params.Arguments["keyword"]
	if keyword == "" {
		keyword = "[keyword]"
	}

	promptText := `Search WhatsApp for "` + keyword + `".

search_messages(query="` + keyword + `")

Show results grouped by chat with timestamps. Use *` + keyword + `* for case-sensitive glob matching, or add from="<JID>" to filter by sender.`

	return mcp.NewGetPromptResult(
		"Search for '"+keyword+"' across all chats",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewTextContent(promptText),
			),
		},
	), nil
}
