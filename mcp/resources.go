package mcp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"whatsapp-mcp/paths"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerResources defines all MCP resources for documentation.
func (m *MCPServer) registerResources() {
	// consolidated usage guide
	m.server.AddResource(
		mcp.NewResource(
			"whatsapp://guide",
			"WhatsApp MCP Guide",
			mcp.WithResourceDescription("Usage guide: workflows, JID formats, tool scope, search patterns, and history loading"),
			mcp.WithMIMEType("text/markdown"),
		),
		m.handleGuide,
	)

	// dynamic media resource template
	m.server.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"whatsapp://media/{message_id}",
			"WhatsApp Media File",
			mcp.WithTemplateDescription("Access media file from a WhatsApp message (image, video, audio, document)"),
		),
		m.handleMediaResource,
	)
}

// handleGuide returns the consolidated WhatsApp MCP usage guide.
func (m *MCPServer) handleGuide(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	guide := `# WhatsApp MCP Guide

## Core workflow
Most operations need a **JID** (WhatsApp's internal ID). Always get one via ` + "`find_chat`" + ` first — never construct them by hand.

- ` + "`find_chat(search=\"name\")`" + ` returns matching chats with their JIDs. Supports wildcards (see Search patterns).
- Names can duplicate or change; JIDs are permanent and unique.
- If ` + "`find_chat`" + ` returns nothing, the contact may not be saved locally or hasn't messaged you yet.

## JID formats
- **DM**: ` + "`<phone>@s.whatsapp.net`" + ` — digits only, country code, no ` + "`+`" + ` or spaces (e.g. ` + "`5511999999999@s.whatsapp.net`" + `)
- **Group**: ` + "`<id>@g.us`" + `
- **Channel**: ` + "`<id>@newsletter`" + `

## Tool scope (these are NOT interchangeable)
- ` + "`get_chat_messages(chat_jid=X)`" + ` — messages from **one chat only**
- ` + "`search_messages(from=JID)`" + ` — messages from **one person across ALL chats** (their DM with you + every group they're in)

To find everything someone has ever said to you anywhere, use ` + "`search_messages(from=JID)`" + ` and **omit ` + "`query`" + ` entirely** (don't pass an empty string). Combine ` + "`from`" + ` + ` + "`query`" + ` to filter their messages by keyword.

## Search patterns
Default: **case-insensitive substring** match. The moment your query contains ` + "`*`" + `, ` + "`?`" + `, or ` + "`[...]`" + `, it switches to GLOB and becomes **case-sensitive**.

| Pattern | Meaning |
|---|---|
| ` + "`*`" + ` | zero or more characters |
| ` + "`?`" + ` | exactly one character |
| ` + "`[abc]`" + ` | one of a, b, c |
| ` + "`[a-z]`" + ` | range |
| ` + "`[^abc]`" + ` | NOT in set |

Trick: ` + "`[Tt]odo`" + ` matches both cases when you need wildcards elsewhere. Avoid bare ` + "`*`" + ` — it matches everything.

## Loading older history
Local DB only holds what's been synced. To pull older messages from WhatsApp's servers:
1. ` + "`load_more_messages(chat_jid=X, count=N)`" + `
2. Then ` + "`get_chat_messages(chat_jid=X)`" + ` to read them

## Notes
- Timestamps are shown in the server timezone (` + m.timezone.String() + `).
- Use ` + "`limit`" + ` and ` + "`offset`" + ` for pagination on large result sets.
`

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "whatsapp://guide",
			MIMEType: "text/markdown",
			Text:     guide,
		},
	}, nil
}

// handleMediaResource handles dynamic media resource requests.
func (m *MCPServer) handleMediaResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	uri := req.Params.URI

	var messageID string
	if req.Params.Arguments != nil {
		if messageIDs, ok := req.Params.Arguments["message_id"].([]string); ok && len(messageIDs) > 0 {
			messageID = messageIDs[0]
		}
	}

	if messageID == "" {
		return nil, errors.New("invalid message id")
	}

	// get media metadata
	meta, err := m.mediaStore.GetMediaMetadata(messageID)
	if err != nil || meta == nil {
		return nil, fmt.Errorf("media not found for message: %s", messageID)
	}

	// check download status
	if meta.DownloadStatus != "downloaded" {
		return nil, fmt.Errorf("media not downloaded (status: %s). Enable auto-download or download manually.", meta.DownloadStatus)
	}

	// sanitize and validate file path to prevent directory traversal
	cleanPath := filepath.Clean(meta.FilePath)
	if strings.Contains(cleanPath, "..") {
		return nil, errors.New("invalid file path: path traversal detected")
	}

	// construct full file path
	fullPath := paths.GetMediaPath(cleanPath)

	// get absolute paths for security validation
	mediaDir, err := filepath.Abs(paths.DataMediaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve media directory: %w", err)
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	// ensure the resolved path is within the media directory
	if !strings.HasPrefix(absPath, mediaDir+string(filepath.Separator)) && absPath != mediaDir {
		return nil, errors.New("invalid file path: outside media directory")
	}

	// verify file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		m.log.Printf("Media file not found at path: %s", absPath)
		return nil, errors.New("media file not found")
	}

	// read the actual file data (use validated absolute path)
	fileData, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read media file: %w", err)
	}

	// encode to base64 for transmission
	encodedData := base64.StdEncoding.EncodeToString(fileData)

	// return the file as a blob so AI assistants can view it
	return []mcp.ResourceContents{
		mcp.BlobResourceContents{
			URI:      uri,
			MIMEType: meta.MimeType,
			Blob:     encodedData,
		},
	}, nil
}
