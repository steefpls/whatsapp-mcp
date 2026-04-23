package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// registerTools defines all MCP tools available to clients.
func (m *MCPServer) registerTools() {
	// 1. list all chats
	m.server.AddTool(
		mcp.NewTool("list_chats",
			mcp.WithDescription("List WhatsApp conversations ordered by most recent activity."),
			mcp.WithNumber("limit",
				mcp.Description("maximum number of chats to return (default: 50, max: 100)"),
			),
		),
		m.handleListChats,
	)

	// 2. get messages from specific chat
	m.server.AddTool(
		mcp.NewTool("get_chat_messages",
			mcp.WithDescription("Retrieve messages from one chat. Supports filtering by sender and timestamp pagination."),
			mcp.WithString("chat_jid",
				mcp.Required(),
				mcp.Description("chat JID (WhatsApp identifier) from find_chat or list_chats"),
			),
			mcp.WithNumber("limit",
				mcp.Description("maximum number of messages to return (default: 50, max: 200)"),
			),
			mcp.WithString("before_timestamp",
				mcp.Description("get messages before this timestamp (ISO 8601 format)"),
			),
			mcp.WithString("after_timestamp",
				mcp.Description("get messages after this timestamp (ISO 8601 format)"),
			),
			mcp.WithString("from",
				mcp.Description("filter messages by sender JID (e.g., for filtering one person's messages in a group chat)"),
			),
			mcp.WithNumber("offset",
				mcp.Description("number of messages to skip for pagination (default: 0)"),
			),
		),
		m.handleGetChatMessages,
	)

	// 3. search messages by text
	m.server.AddTool(
		mcp.NewTool("search_messages",
			mcp.WithDescription("Search messages across all chats by text or sender. Supports glob wildcards (*, ?, [abc])."),
			mcp.WithString("query",
				mcp.Description("text pattern to search for (optional: can be omitted when using only 'from' parameter)"),
			),
			mcp.WithString("from",
				mcp.Description("filter by sender JID to find all messages from a specific person across all chats"),
			),
			mcp.WithNumber("limit",
				mcp.Description("maximum number of results to return (default: 50, max: 200)"),
			),
		),
		m.handleSearchMessages,
	)

	// 4. find chat by name or JID
	m.server.AddTool(
		mcp.NewTool("find_chat",
			mcp.WithDescription("Find chats by name or JID. Supports glob wildcards."),
			mcp.WithString("search",
				mcp.Required(),
				mcp.Description("search pattern (supports wildcards: *, ?, [abc])"),
			),
		),
		m.handleFindChat,
	)

	// 5. send message
	m.server.AddTool(
		mcp.NewTool("send_message",
			mcp.WithDescription("Send a text message to a WhatsApp chat (DM or group)."),
			mcp.WithString("chat_jid",
				mcp.Required(),
				mcp.Description("recipient chat JID from find_chat or list_chats"),
			),
			mcp.WithString("text",
				mcp.Required(),
				mcp.Description("message text to send"),
			),
		),
		m.handleSendMessage,
	)

	// 5b. send file (image, video, audio, document)
	m.server.AddTool(
		mcp.NewTool("send_file",
			mcp.WithDescription("Send a local file to a chat. Auto-detects media type; use as_document=true to force generic document."),
			mcp.WithString("chat_jid",
				mcp.Required(),
				mcp.Description("recipient chat JID from find_chat or list_chats"),
			),
			mcp.WithString("path",
				mcp.Required(),
				mcp.Description("absolute path to the file on disk"),
			),
			mcp.WithString("caption",
				mcp.Description("optional caption (image/video/document only — ignored for audio)"),
			),
			mcp.WithBoolean("as_document",
				mcp.Description("if true, send as a generic document regardless of MIME type (default: false)"),
			),
		),
		m.handleSendFile,
	)

	// 5c. send emoji reaction to a message
	m.server.AddTool(
		mcp.NewTool("send_reaction",
			mcp.WithDescription("React to a message with an emoji. Pass an empty emoji to remove a previous reaction."),
			mcp.WithString("message_id",
				mcp.Required(),
				mcp.Description("ID of the message to react to (from get_chat_messages or search_messages)"),
			),
			mcp.WithString("emoji",
				mcp.Required(),
				mcp.Description("emoji to react with (e.g., \"👍\", \"❤️\"). Empty string removes the reaction."),
			),
		),
		m.handleSendReaction,
	)

	// 6. load more messages on-demand
	m.server.AddTool(
		mcp.NewTool("load_more_messages",
			mcp.WithDescription("Fetch older messages from WhatsApp servers when not yet in the local database."),
			mcp.WithString("chat_jid",
				mcp.Required(),
				mcp.Description("chat JID to fetch history for"),
			),
			mcp.WithNumber("count",
				mcp.Description("number of messages to fetch (default: 50, max: 200)"),
			),
			mcp.WithBoolean("wait_for_sync",
				mcp.Description("if true (default), waits for messages to arrive before returning. If false, messages load in background."),
			),
		),
		m.handleLoadMoreMessages,
	)

	// 7. get my info
	m.server.AddTool(
		mcp.NewTool("get_my_info",
			mcp.WithDescription("Get your own WhatsApp profile (JID, name, status, picture URL)."),
		),
		m.handleGetMyInfo,
	)

	// 8. add trusted user for @claude trigger
	m.server.AddTool(
		mcp.NewTool("add_trusted_user",
			mcp.WithDescription("Allow a user to trigger @claude in messages."),
			mcp.WithString("jid",
				mcp.Required(),
				mcp.Description("WhatsApp JID of the user to trust (e.g., 6591234567@s.whatsapp.net)"),
			),
		),
		m.handleAddTrustedUser,
	)

	// 9. remove trusted user
	m.server.AddTool(
		mcp.NewTool("remove_trusted_user",
			mcp.WithDescription("Revoke a user's ability to trigger @claude."),
			mcp.WithString("jid",
				mcp.Required(),
				mcp.Description("WhatsApp JID of the user to remove from trusted list"),
			),
		),
		m.handleRemoveTrustedUser,
	)

	// 10. list trusted users
	m.server.AddTool(
		mcp.NewTool("list_trusted_users",
			mcp.WithDescription("List users authorized to trigger @claude."),
		),
		m.handleListTrustedUsers,
	)
}
