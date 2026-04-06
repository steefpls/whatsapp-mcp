package claude

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"whatsapp-mcp/storage"
)

// thinkingMessages are randomly selected when Claude starts processing a request.
var thinkingMessages = []string{
	// Classic
	"🤖 Claude is thinking...",
	"🧠 Claude is on it...",
	"💭 Claude is pondering...",
	"🤔 Claude is processing...",
	"✨ Claude is doing its thing...",
	// Cooking
	"⚡ Claude is cooking something up...",
	"🍳 Claude is in the kitchen...",
	"👨‍🍳 Chef Claude is preparing your order...",
	"🔥 Claude is heating up...",
	// Vibes
	"🌀 Claude has entered the chat...",
	"🎯 Claude is locked in...",
	"🛠️ Claude is working on it...",
	"🔮 Claude is consulting the oracle...",
	"🚀 Claude is launching...",
	"⏳ Claude is buffering... just kidding, actually thinking",
	"🎵 Claude is vibing and computing...",
	"🧪 Claude is running experiments...",
	"📡 Claude is downloading intelligence...",
	"🌊 Claude is riding the wave of tokens...",
	// Dramatic
	"⚔️ Claude is battling the void for answers...",
	"🏋️ Claude is lifting heavy thoughts...",
	"🧙 Claude is casting a spell...",
	"🎭 Claude is rehearsing a response...",
	"🕵️ Claude is investigating...",
	"🔬 Claude is analyzing the situation...",
	// Silly
	"🐙 Claude is using all eight brain tentacles...",
	"☕ Claude is sipping coffee and contemplating...",
	"🎰 Claude is rolling the neural dice...",
	"🧩 Claude is assembling the puzzle pieces...",
	"📖 Claude is reading the ancient scrolls...",
	"🌌 Claude is traversing the latent space...",
	"🎪 Claude is performing inference acrobatics...",
	"🏄 Claude is surfing the attention layers...",
	"🧲 Claude is attracting relevant information...",
	"💡 Claude just had a thought... hold on...",
	"🦾 Claude is flexing its parameters...",
	"🛸 Claude is phoning the mothership...",
	"🎲 Claude is sampling from the probability gods...",
	"🧬 Claude is decoding your request at the molecular level...",
	"🌋 Claude is erupting with ideas...",
	"🎸 Claude is shredding through the computation...",
	"🦊 Claude is being sly and clever...",
	"🍜 Claude is slurping up context...",
	"⛏️ Claude is mining for the perfect response...",
	"🪄 Claude is waving the magic wand...",
	// More silly
	"🐢 Claude is slow-cooking this one for quality...",
	"🧈 Claude is buttering up a response...",
	"🫠 Claude is melting into deep thought...",
	"🐝 Claude is buzzing through the hive mind...",
	"🦑 Claude is ink-jetting a response...",
	"🍕 Claude is slicing up some hot takes...",
	"🧃 Claude is juicing the neurons...",
	"🎳 Claude is going for a strike...",
	"🪃 Claude's thoughts are coming back around...",
	"🌮 Claude is wrapping up something good...",
	"🐸 Claude says: ribbit... I mean, processing...",
	"🎣 Claude is fishing for the right words...",
	"🥊 Claude is sparring with the problem...",
	"🧊 Claude is breaking the ice...",
	"🎨 Claude is painting a masterpiece response...",
	"🔧 Claude is tinkering under the hood...",
	"🗿 Claude is channeling ancient wisdom...",
	"🦥 Claude is... wait for it...",
	"🎯 Claude is aiming for perfection...",
	"🏗️ Claude is constructing a response...",
	// Nerdy
	"📊 Claude is running the numbers...",
	"🧮 Claude is doing the math...",
	"💾 Claude is loading response.exe...",
	"🖥️ Claude.exe is not responding... jk, almost done",
	"📝 Claude is drafting revision 42...",
	"🔋 Claude is charging up...",
	"🌐 Claude is querying the multiverse...",
	"🛰️ Claude is establishing satellite uplink...",
	"⚙️ Claude is turning the gears...",
	"🧫 Claude is growing a response in the petri dish...",
	// Dramatic pt2
	"🏔️ Claude is scaling the mountain of knowledge...",
	"🌪️ Claude is caught in a brainstorm...",
	"🎆 Claude is about to have a eureka moment...",
	"🦅 Claude is soaring through the data...",
	"🏊 Claude is diving deep...",
	"⚗️ Claude is distilling pure wisdom...",
	"🗡️ Claude is cutting through the noise...",
	"🔭 Claude is stargazing for answers...",
	"🧭 Claude is navigating the response space...",
	"🗺️ Claude is charting a course...",
	// Pop culture-ish
	"🪐 Claude is in another dimension rn, hold tight...",
	"👾 Claude has entered the matrix...",
	"🫡 Claude is on a mission...",
	"🦸 Claude is suiting up...",
	"🎬 Claude is directing the perfect scene...",
	"🏎️ Claude is in the fast lane...",
	"🎮 Claude is speed-running this...",
	"🪂 Claude is parachuting in with answers...",
	"🧗 Claude is climbing toward enlightenment...",
	"🎻 Claude is composing a symphony of words...",
	// Wholesome
	"💪 Claude believes in this response...",
	"🌱 Claude is growing an idea...",
	"🌈 Claude is chasing the rainbow of insight...",
	"🤗 Claude is putting thought and care into this...",
	"🍵 Claude is brewing something warm...",
	"🎁 Claude is wrapping up a gift for you...",
	// Chaotic
	"🐉 Claude has awakened...",
	"💥 Claude is detonating brain cells...",
	"🌶️ Claude is making it extra spicy...",
	"🤯 Claude's mind is blown, reassembling...",
	"🫧 Claude is blowing thought bubbles...",
	"🍿 Claude is making popcorn while it thinks...",
	"👀 Claude saw your message and is LOCKED IN...",
	"🔥 Claude is on fire (metaphorically)...",
	"💨 Claude is speed-thinking...",
	"🐋 Claude is whale-processing your request...",
}

// signature is appended to every Claude response for attribution.
const signature = "\n\n— _sent by @claude_ 🤖"

// Trigger handles spawning headless Claude Code CLI instances for @claude mentions.
type Trigger struct {
	config      TriggerConfig
	sendMsg     func(ctx context.Context, chatJID string, text string) error
	sendMsgID   func(ctx context.Context, chatJID string, text string) (string, error)
	revokeMsg   func(ctx context.Context, chatJID string, messageID string) error
	editMsg     func(ctx context.Context, chatJID string, messageID string, newText string) error
	getHistory  func(chatJID string, limit int, offset int) ([]storage.MessageWithNames, error)
	isTrusted   func(jid string) (bool, error)
	log         *log.Logger
}

// NewTrigger creates a new Claude trigger.
func NewTrigger(
	config TriggerConfig,
	sendMsg func(ctx context.Context, chatJID string, text string) error,
	sendMsgID func(ctx context.Context, chatJID string, text string) (string, error),
	revokeMsg func(ctx context.Context, chatJID string, messageID string) error,
	editMsg func(ctx context.Context, chatJID string, messageID string, newText string) error,
	getHistory func(chatJID string, limit int, offset int) ([]storage.MessageWithNames, error),
	isTrusted func(jid string) (bool, error),
) *Trigger {
	return &Trigger{
		config:     config,
		sendMsg:    sendMsg,
		sendMsgID:  sendMsgID,
		revokeMsg:  revokeMsg,
		editMsg:    editMsg,
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

	// send randomized acknowledgment (track ID for later deletion)
	ack := thinkingMessages[rand.Intn(len(thinkingMessages))]
	ackID, ackErr := t.sendMsgID(ctx, chatJID, ack)
	if ackErr != nil {
		t.log.Printf("[CLAUDE] Failed to send ack to %s: %v", chatJID, ackErr)
	}

	// fetch last 100 messages for context
	messages, err := t.getHistory(chatJID, 100, 0)
	if err != nil {
		t.log.Printf("[CLAUDE] Failed to get history for %s: %v", chatJID, err)
		t.sendMsg(ctx, chatJID, "Sorry, I couldn't load the conversation history."+signature)
		return
	}

	// build prompt and execute
	prompt := t.buildPrompt(chatJID, messages, text, senderName)
	output, err := t.execClaude(ctx, prompt)
	if err != nil {
		t.log.Printf("[CLAUDE] CLI execution failed for %s: %v", chatJID, err)
		t.sendMsg(ctx, chatJID, "Sorry, something went wrong while processing your request."+signature)
		return
	}

	// edit the thinking message with the actual response
	output = strings.TrimSpace(output)
	if output != "" {
		response := output + signature
		if ackErr == nil && ackID != "" {
			if err := t.editMsg(ctx, chatJID, ackID, response); err != nil {
				t.log.Printf("[CLAUDE] Failed to edit ack message %s, sending new: %v", ackID, err)
				// fallback: send as new message if edit fails
				t.sendMsg(ctx, chatJID, response)
			}
		} else {
			t.sendMsg(ctx, chatJID, response)
		}
	}
}

// buildPrompt constructs the prompt for the headless Claude instance.
func (t *Trigger) buildPrompt(chatJID string, messages []storage.MessageWithNames, triggerText, senderName string) string {
	var b strings.Builder

	b.WriteString("You are responding to a WhatsApp message. A user mentioned @claude in a WhatsApp chat.\n\n")

	// determine the chat name from messages for context
	chatName := ""
	if len(messages) > 0 {
		chatName = messages[0].ChatName
	}
	if chatName != "" {
		fmt.Fprintf(&b, "This chat is with: %s\n\n", chatName)
	}

	b.WriteString("Here are the last 100 messages from this chat for context:\n")
	b.WriteString("Messages from \"Steve\" are from the WhatsApp account owner.\n\n")

	// messages come newest-first from DB, display oldest-first
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		var sender string
		if msg.IsFromMe {
			sender = "Steve"
		} else {
			sender = msg.SenderContactName
			if sender == "" {
				sender = msg.SenderPushName
			}
			if sender == "" {
				sender = msg.SenderJID
			}
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
