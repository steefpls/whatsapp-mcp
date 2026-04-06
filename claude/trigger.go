package claude

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"whatsapp-mcp/storage"
)

// Trigger handles spawning headless Claude Code CLI instances for @claude mentions.
type Trigger struct {
	config     TriggerConfig
	sendMsg    func(ctx context.Context, chatJID string, text string) error
	getHistory func(chatJID string, limit int, offset int) ([]storage.MessageWithNames, error)
	isTrusted  func(jid string) (bool, error)
	log        *log.Logger
}

// NewTrigger creates a new Claude trigger.
func NewTrigger(
	config TriggerConfig,
	sendMsg func(ctx context.Context, chatJID string, text string) error,
	getHistory func(chatJID string, limit int, offset int) ([]storage.MessageWithNames, error),
	isTrusted func(jid string) (bool, error),
) *Trigger {
	return &Trigger{
		config:     config,
		sendMsg:    sendMsg,
		getHistory: getHistory,
		isTrusted:  isTrusted,
		log:        log.Default(),
	}
}

// HandleTrigger processes an @claude mention. Runs synchronously — caller should invoke in a goroutine.
func (t *Trigger) HandleTrigger(ctx context.Context, chatJID, senderJID, text, senderName string, isFromMe bool) {
	// access control: owner always allowed, others must be trusted
	if !isFromMe {
		trusted, err := t.isTrusted(senderJID)
		if err != nil {
			t.log.Printf("[CLAUDE] Failed to check trusted status for %s: %v", senderJID, err)
			return
		}
		if !trusted {
			rejectMsg := fmt.Sprintf("Sorry %s, you're not authorized to use @claude. Ask the owner to add you with the add_trusted_user tool.", senderName)
			if err := t.sendMsg(ctx, chatJID, rejectMsg); err != nil {
				t.log.Printf("[CLAUDE] Failed to send rejection to %s: %v", chatJID, err)
			}
			return
		}
	}

	// send acknowledgment
	if err := t.sendMsg(ctx, chatJID, "Got it, thinking..."); err != nil {
		t.log.Printf("[CLAUDE] Failed to send ack to %s: %v", chatJID, err)
	}

	// fetch last 10 messages for context
	messages, err := t.getHistory(chatJID, 10, 0)
	if err != nil {
		t.log.Printf("[CLAUDE] Failed to get history for %s: %v", chatJID, err)
		t.sendMsg(ctx, chatJID, "Sorry, I couldn't load the conversation history.")
		return
	}

	// build prompt and execute
	prompt := t.buildPrompt(chatJID, messages, text, senderName)
	output, err := t.execClaude(ctx, prompt)
	if err != nil {
		t.log.Printf("[CLAUDE] CLI execution failed for %s: %v", chatJID, err)
		t.sendMsg(ctx, chatJID, "Sorry, something went wrong while processing your request.")
		return
	}

	// send response if Claude produced text output
	// (Claude may have already sent messages via the WhatsApp MCP send_message tool)
	output = strings.TrimSpace(output)
	if output != "" {
		if err := t.sendMsg(ctx, chatJID, output); err != nil {
			t.log.Printf("[CLAUDE] Failed to send response to %s: %v", chatJID, err)
		}
	}
}

// buildPrompt constructs the prompt for the headless Claude instance.
func (t *Trigger) buildPrompt(chatJID string, messages []storage.MessageWithNames, triggerText, senderName string) string {
	var b strings.Builder

	b.WriteString("You are responding to a WhatsApp message. A user mentioned @claude in a WhatsApp chat.\n\n")
	b.WriteString("Here are the last messages from this chat for context:\n\n")

	// messages come newest-first from DB, display oldest-first
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		sender := msg.SenderContactName
		if sender == "" {
			sender = msg.SenderPushName
		}
		if sender == "" {
			sender = msg.SenderJID
		}
		if msg.IsFromMe {
			sender = "Me (WhatsApp owner)"
		}

		fmt.Fprintf(&b, "[%s] %s: %s\n",
			msg.Timestamp.Format("2006-01-02 15:04:05"),
			sender,
			msg.Text)
	}

	b.WriteString("\n---\n\n")
	fmt.Fprintf(&b, "The message that triggered you (from %s):\n%s\n\n", senderName, triggerText)
	fmt.Fprintf(&b, "The chat JID to reply to is: %s\n\n", chatJID)
	b.WriteString("Respond to the user's request. Be helpful, concise, and conversational.\n")
	b.WriteString("IMPORTANT: Do NOT include the literal text \"@claude\" anywhere in your response to avoid re-triggering yourself.\n")
	b.WriteString("IMPORTANT: Do NOT use the send_message tool to reply. Just output your response text directly — it will be automatically sent as a WhatsApp message for you.\n")
	b.WriteString("You may use other WhatsApp tools (search_messages, get_chat_messages, find_chat, etc.) if the user's request requires looking up information.\n")

	return b.String()
}

// writeMCPConfig writes a temporary MCP config file for the spawned Claude instance
// and returns its path. The caller is responsible for cleaning it up.
func (t *Trigger) writeMCPConfig() (string, error) {
	mcpJSON := fmt.Sprintf(
		`{"mcpServers":{"whatsapp":{"type":"http","url":"http://localhost:%s/mcp/%s"}}}`,
		t.config.MCPPort, t.config.MCPAPIKey,
	)

	dir := filepath.Join("data", "tmp")
	os.MkdirAll(dir, 0755)

	f, err := os.CreateTemp(dir, "claude-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp MCP config: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(mcpJSON); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to write MCP config: %w", err)
	}

	return f.Name(), nil
}

// execClaude spawns the Claude CLI and returns its output.
func (t *Trigger) execClaude(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	// write temp MCP config file
	mcpConfigPath, err := t.writeMCPConfig()
	if err != nil {
		return "", err
	}
	defer os.Remove(mcpConfigPath)

	args := []string{
		"--print",
		"--dangerously-skip-permissions",
	}

	if t.config.Model != "" {
		args = append(args, "--model", t.config.Model)
	}
	if t.config.MaxBudget != "" {
		args = append(args, "--max-budget-usd", t.config.MaxBudget)
	}

	// --mcp-config is variadic so it must be last before we pipe the prompt via stdin
	args = append(args, "--mcp-config", mcpConfigPath)

	t.log.Printf("[CLAUDE] Spawning: %s (model=%s, mcp=%s)", t.config.ClaudePath, t.config.Model, mcpConfigPath)

	cmd := exec.CommandContext(ctx, t.config.ClaudePath, args...)
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude CLI failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.String(), nil
}
