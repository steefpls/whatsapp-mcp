package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"whatsapp-mcp/claude"
	"whatsapp-mcp/storage"

	"github.com/mark3labs/mcp-go/mcp"
)

// getDisplayName returns the best available name for a chat
// Priority: ContactName > PushName > JID
func getDisplayName(chat storage.Chat) string {
	if chat.ContactName != "" {
		return chat.ContactName
	}
	if chat.PushName != "" {
		return chat.PushName
	}
	return chat.JID
}

// getSenderDisplayName returns the best available name for a message sender
// Priority: ContactName > PushName > JID
func getSenderDisplayName(msg storage.MessageWithNames) string {
	if msg.SenderContactName != "" {
		return msg.SenderContactName
	}
	if msg.SenderPushName != "" {
		return msg.SenderPushName
	}
	return msg.SenderJID
}

// toLocalTime converts a UTC timestamp to the configured timezone.
func (m *MCPServer) toLocalTime(t time.Time) time.Time {
	return t.In(m.timezone)
}

// formatDateTime formats a timestamp in the configured timezone for date and time display.
func (m *MCPServer) formatDateTime(t time.Time) string {
	return m.toLocalTime(t).Format("2006-01-02 15:04:05")
}

// formatTime formats a timestamp in the configured timezone for time-only display.
func (m *MCPServer) formatTime(t time.Time) string {
	return m.toLocalTime(t).Format("15:04:05")
}

// parseTimestamp converts an ISO 8601 timestamp string to time.Time in the server's timezone.
// It supports the formats: "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02".
func (m *MCPServer) parseTimestamp(timestampStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, timestampStr, m.timezone); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid timestamp format: %s (expected ISO 8601 like '2006-01-02T15:04:05' or '2006-01-02')", timestampStr)
}

// detectPatternType determines whether a search query should use GLOB matching.
// It returns true if the query contains glob wildcards: * ? [
func detectPatternType(query string) bool {
	return strings.ContainsAny(query, "*?[")
}

// formatFileSize converts bytes to a human-readable size string.
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	if bytes >= GB {
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	} else if bytes >= MB {
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	} else if bytes >= KB {
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	}
	return fmt.Sprintf("%d B", bytes)
}

// formatDimensions returns a formatted dimensions string from width and height.
func formatDimensions(width, height *int) string {
	if width != nil && height != nil {
		return fmt.Sprintf("%dx%d", *width, *height)
	}
	return ""
}

// formatDuration converts seconds to MM:SS format.
func formatDuration(seconds *int) string {
	if seconds == nil {
		return ""
	}
	s := *seconds
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// formatMediaLine returns a single compact line describing media metadata.
// Includes the media URI when downloaded, or a status tag otherwise.
func formatMediaLine(meta *storage.MediaMetadata, msgID string) string {
	if meta == nil {
		return ""
	}
	parts := []string{"📎", meta.FileName, formatFileSize(meta.FileSize)}
	if dims := formatDimensions(meta.Width, meta.Height); dims != "" {
		parts = append(parts, dims)
	}
	if dur := formatDuration(meta.Duration); dur != "" {
		parts = append(parts, dur)
	}
	if meta.DownloadStatus == "downloaded" {
		parts = append(parts, "→ whatsapp://media/"+msgID)
	} else {
		parts = append(parts, "["+meta.DownloadStatus+"]")
	}
	return strings.Join(parts, " ")
}

// handleListChats handles the list_chats tool request.
func (m *MCPServer) handleListChats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// get limit parameter with default
	limit := request.GetFloat("limit", 50.0)
	if limit > 100 {
		limit = 100
	}

	// query database
	chats, err := m.store.ListChats(int(limit))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list chats: %v", err)), nil
	}

	// format response: one compact line per chat
	var result strings.Builder
	for i, chat := range chats {
		chatType := "DM"
		if chat.IsGroup {
			chatType = "Group"
		}
		fmt.Fprintf(&result, "%d. %s | %s | %s | %s",
			i+1,
			chatType,
			getDisplayName(chat),
			chat.JID,
			m.formatDateTime(chat.LastMessageTime))
		if chat.UnreadCount > 0 {
			fmt.Fprintf(&result, " | unread:%d", chat.UnreadCount)
		}
		result.WriteString("\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleGetChatMessages handles the get_chat_messages tool request.
func (m *MCPServer) handleGetChatMessages(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// get required chat_jid
	chatJID, err := request.RequireString("chat_jid")
	if err != nil {
		return mcp.NewToolResultError("chat_jid parameter is required"), nil
	}

	// get optional limit
	limit := request.GetFloat("limit", 50.0)
	if limit > 200 {
		limit = 200
	}

	// get optional timestamp filters
	var beforeTime *time.Time
	var afterTime *time.Time

	beforeStr := request.GetString("before_timestamp", "")
	if beforeStr != "" {
		t, err := m.parseTimestamp(beforeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid before_timestamp: %v", err)), nil
		}
		beforeTime = &t
	}

	afterStr := request.GetString("after_timestamp", "")
	if afterStr != "" {
		t, err := m.parseTimestamp(afterStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid after_timestamp: %v", err)), nil
		}
		afterTime = &t
	}

	// get optional sender filter
	senderJID := request.GetString("from", "")

	// query database
	var messages []storage.MessageWithNames

	if beforeTime != nil || afterTime != nil || senderJID != "" {
		// use new filtered method
		messages, err = m.store.GetChatMessagesWithNamesFiltered(
			chatJID,
			int(limit),
			beforeTime,
			afterTime,
			senderJID,
		)
	} else {
		// backward compatibility: use offset if no timestamp filters
		offset := request.GetFloat("offset", 0.0)
		messages, err = m.store.GetChatMessagesWithNames(chatJID, int(limit), int(offset))
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get messages: %v", err)), nil
	}

	// collect message IDs for batch edit-history lookup
	var msgIDs []string
	for _, msg := range messages {
		msgIDs = append(msgIDs, msg.ID)
	}
	editHistories, _ := m.store.GetEditHistoryForMessages(msgIDs)

	// format response: oldest first, compact lines
	var result strings.Builder
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		// skip @claude trigger ack ghosts
		if msg.IsFromMe && claude.IsThinkingMessage(msg.Text) {
			continue
		}
		sender := getSenderDisplayName(msg)
		direction := "←"
		if msg.IsFromMe {
			direction = "→"
			sender = "You"
		}

		text := msg.Text
		if text == "" && msg.MediaMetadata != nil {
			// media-only message: put media on the main line
			text = formatMediaLine(msg.MediaMetadata, msg.ID)
		}
		fmt.Fprintf(&result, "%s %s %s: %s\n", m.formatTime(msg.Timestamp), direction, sender, text)

		// captioned media: put media on an indented line below
		if msg.Text != "" && msg.MediaMetadata != nil {
			fmt.Fprintf(&result, "    %s\n", formatMediaLine(msg.MediaMetadata, msg.ID))
		}

		// show edit history if this message was edited
		if edits, ok := editHistories[msg.ID]; ok && len(edits) > 0 {
			for j, e := range edits {
				fmt.Fprintf(&result, "    ✏️ edit %d at %s: \"%s\" → \"%s\"\n", j+1, m.formatTime(e.EditedAt), e.OldText, e.NewText)
			}
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleSearchMessages handles the search_messages tool request.
func (m *MCPServer) handleSearchMessages(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// get query (can be empty when using 'from' parameter)
	query := request.GetString("query", "")

	// get optional limit
	limit := request.GetFloat("limit", 50.0)
	if limit > 200 {
		limit = 200
	}

	// get optional sender filter
	senderJID := request.GetString("from", "")

	// validate: must have either query or from
	if query == "" && senderJID == "" {
		return mcp.NewToolResultError("must provide either 'query' (text to search) or 'from' (sender JID) or both"), nil
	}

	// detect pattern type
	useGlob := detectPatternType(query)

	// search database
	messages, err := m.store.SearchMessagesWithNamesFiltered(query, useGlob, senderJID, int(limit))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// group messages by chat to avoid repeating the chat JID per message
	sort.SliceStable(messages, func(i, j int) bool {
		if messages[i].ChatJID != messages[j].ChatJID {
			return messages[i].ChatJID < messages[j].ChatJID
		}
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	var result strings.Builder
	currentChat := ""
	for _, msg := range messages {
		// skip @claude trigger ack ghosts
		if msg.IsFromMe && claude.IsThinkingMessage(msg.Text) {
			continue
		}
		if msg.ChatJID != currentChat {
			if currentChat != "" {
				result.WriteString("\n")
			}
			fmt.Fprintf(&result, "[%s]\n", msg.ChatJID)
			currentChat = msg.ChatJID
		}

		sender := getSenderDisplayName(msg)
		if msg.IsFromMe {
			sender = "You"
		}

		text := msg.Text
		if text == "" && msg.MediaMetadata != nil {
			text = formatMediaLine(msg.MediaMetadata, msg.ID)
		}
		fmt.Fprintf(&result, "  %s %s: %s\n", m.formatDateTime(msg.Timestamp), sender, text)
		if msg.Text != "" && msg.MediaMetadata != nil {
			fmt.Fprintf(&result, "      %s\n", formatMediaLine(msg.MediaMetadata, msg.ID))
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleFindChat handles the find_chat tool request.
func (m *MCPServer) handleFindChat(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// get required search parameter
	search, err := request.RequireString("search")
	if err != nil {
		return mcp.NewToolResultError("search parameter is required"), nil
	}

	// detect pattern type
	useGlob := detectPatternType(search)

	// search chats in database
	chats, err := m.store.SearchChatsFiltered(search, useGlob, 100)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search chats: %v", err)), nil
	}

	// format response: one compact line per chat
	var result strings.Builder
	for i, chat := range chats {
		chatType := "DM"
		if chat.IsGroup {
			chatType = "Group"
		}
		fmt.Fprintf(&result, "%d. %s | %s | %s\n",
			i+1, chatType, getDisplayName(chat), chat.JID)
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleSendMessage handles the send_message tool request.
func (m *MCPServer) handleSendMessage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// get required parameters
	chatJID, err := request.RequireString("chat_jid")
	if err != nil {
		return mcp.NewToolResultError("chat_jid parameter is required"), nil
	}

	text, err := request.RequireString("text")
	if err != nil {
		return mcp.NewToolResultError("text parameter is required"), nil
	}

	// check WhatsApp connection
	if !m.wa.IsLoggedIn() {
		return mcp.NewToolResultError("WhatsApp is not connected"), nil
	}

	// send message
	err = m.wa.SendTextMessage(ctx, chatJID, text)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to send message: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message sent successfully to %s", chatJID)), nil
}

// handleSendFile handles the send_file tool request.
func (m *MCPServer) handleSendFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chatJID, err := request.RequireString("chat_jid")
	if err != nil {
		return mcp.NewToolResultError("chat_jid parameter is required"), nil
	}

	filePath, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	caption := request.GetString("caption", "")
	asDocument := request.GetBool("as_document", false)

	if !m.wa.IsLoggedIn() {
		return mcp.NewToolResultError("WhatsApp is not connected"), nil
	}

	msgID, kind, err := m.wa.SendFile(ctx, chatJID, filePath, caption, asDocument)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to send file: %v", err)), nil
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Sent %s to %s (message ID: %s)", kind, chatJID, msgID),
	), nil
}

// handleLoadMoreMessages handles the load_more_messages tool request.
func (m *MCPServer) handleLoadMoreMessages(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// get required chat_jid
	chatJID, err := request.RequireString("chat_jid")
	if err != nil {
		return mcp.NewToolResultError("chat_jid parameter is required"), nil
	}

	// get optional count (default 50, max 200)
	count := int(request.GetFloat("count", 50.0))
	if count > 200 {
		count = 200
	}
	if count < 1 {
		count = 1
	}

	// get optional wait_for_sync (default true)
	waitForSync := request.GetBool("wait_for_sync", true)

	// check WhatsApp connection
	if !m.wa.IsLoggedIn() {
		return mcp.NewToolResultError("WhatsApp is not connected"), nil
	}

	// request history sync
	messages, err := m.wa.RequestHistorySync(ctx, chatJID, count, waitForSync)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load messages: %v", err)), nil
	}

	// format response
	var result strings.Builder

	if waitForSync {
		fmt.Fprintf(&result, "Loaded %d messages:\n", len(messages))
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			// skip @claude trigger ack ghosts
			if msg.IsFromMe && claude.IsThinkingMessage(msg.Text) {
				continue
			}
			sender := getSenderDisplayName(msg)
			direction := "←"
			if msg.IsFromMe {
				direction = "→"
				sender = "You"
			}

			text := msg.Text
			if text == "" && msg.MediaMetadata != nil {
				text = formatMediaLine(msg.MediaMetadata, msg.ID)
			}
			fmt.Fprintf(&result, "%s %s %s: %s\n", m.formatTime(msg.Timestamp), direction, sender, text)
			if msg.Text != "" && msg.MediaMetadata != nil {
				fmt.Fprintf(&result, "    %s\n", formatMediaLine(msg.MediaMetadata, msg.ID))
			}
		}
	} else {
		fmt.Fprintf(&result, "History sync requested (%d messages). Use get_chat_messages once loaded.", count)
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleGetMyInfo handles the get_my_info tool request.
func (m *MCPServer) handleGetMyInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// check WhatsApp connection
	if !m.wa.IsLoggedIn() {
		return mcp.NewToolResultError("WhatsApp is not connected"), nil
	}

	// get user info
	myInfo, err := m.wa.GetMyInfo(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get user info: %v", err)), nil
	}

	// format response
	var result strings.Builder
	fmt.Fprintf(&result, "Your WhatsApp Profile:\n\n")
	fmt.Fprintf(&result, "JID: %s\n", myInfo.JID)

	if myInfo.PushName != "" {
		fmt.Fprintf(&result, "Display Name: %s\n", myInfo.PushName)
	}

	if myInfo.Status != "" {
		fmt.Fprintf(&result, "Status/Bio: %s\n", myInfo.Status)
	} else {
		fmt.Fprintf(&result, "Status/Bio: (not set)\n")
	}

	if myInfo.BusinessName != "" {
		fmt.Fprintf(&result, "Business Name: %s\n", myInfo.BusinessName)
	}

	if myInfo.PictureURL != "" {
		fmt.Fprintf(&result, "\nProfile Picture:\n")
		fmt.Fprintf(&result, "  Picture ID: %s\n", myInfo.PictureID)
		fmt.Fprintf(&result, "  URL: %s\n", myInfo.PictureURL)
	} else {
		fmt.Fprintf(&result, "\nProfile Picture: (not set)\n")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// handleAddTrustedUser handles the add_trusted_user tool request.
func (m *MCPServer) handleAddTrustedUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jid, err := request.RequireString("jid")
	if err != nil {
		return mcp.NewToolResultError("jid parameter is required"), nil
	}

	if err := m.trustedUserStore.AddTrustedUser(jid); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add trusted user: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Added %s to trusted users list. They can now trigger @claude in messages.", jid)), nil
}

// handleRemoveTrustedUser handles the remove_trusted_user tool request.
func (m *MCPServer) handleRemoveTrustedUser(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jid, err := request.RequireString("jid")
	if err != nil {
		return mcp.NewToolResultError("jid parameter is required"), nil
	}

	if err := m.trustedUserStore.RemoveTrustedUser(jid); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to remove trusted user: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Removed %s from trusted users list.", jid)), nil
}

// handleListTrustedUsers handles the list_trusted_users tool request.
func (m *MCPServer) handleListTrustedUsers(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	users, err := m.trustedUserStore.ListTrustedUsers()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list trusted users: %v", err)), nil
	}

	var result strings.Builder
	if len(users) == 0 {
		result.WriteString("No trusted users configured. Use add_trusted_user to allow users to trigger @claude.")
	} else {
		fmt.Fprintf(&result, "Trusted users (%d):\n\n", len(users))
		for i, user := range users {
			fmt.Fprintf(&result, "%d. %s (added: %s)\n", i+1, user.JID, m.formatDateTime(user.AddedAt))
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}
