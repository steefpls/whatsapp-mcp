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
	"time"

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

// signatures are randomly selected and appended to every Claude response for attribution.
var signatures = []string{
	"\n\n— _sent by @claude_ 🤖",
	"\n\n— _@claude out_ ✌️",
	"\n\n— _your friendly neighborhood @claude_ 🕸️",
	"\n\n— _@claude, at your service_ 🎩",
	"\n\n— _powered by @claude_ ⚡",
	"\n\n— _@claude was here_ 🖊️",
	"\n\n— _beep boop, @claude_ 🤖",
	"\n\n— _@claude, signing off_ 🫡",
	"\n\n— _this message brought to you by @claude_ 📡",
	"\n\n— _@claude dropping knowledge_ 🎤",
	"\n\n— _from the desk of @claude_ 📝",
	"\n\n— _@claude, over and out_ 📻",
	"\n\n— _delivered by @claude express_ 📦",
	"\n\n— _@claude has spoken_ 🗿",
	"\n\n— _courtesy of @claude_ 🎁",
	"\n\n— _@claude, reporting for duty_ 🫡",
	"\n\n— _transmitted by @claude_ 📡",
	"\n\n— _@claude mic drop_ 🎤⬇️",
	"\n\n— _crafted with care by @claude_ ✨",
	"\n\n— _@claude, peace out_ ✌️🤖",
	// More chaos
	"\n\n— _@claude left the building_ 🏃💨",
	"\n\n— _sent from my @claude_ 📱",
	"\n\n— _@claude: task failed successfully_ ✅",
	"\n\n— _@claude, your AI butler_ 🧐",
	"\n\n— _forged in the fires of @claude_ 🔥",
	"\n\n— _@claude v∞.0_ 🔄",
	"\n\n— _@claude: I came, I saw, I responded_ 🏛️",
	"\n\n— _brewed fresh by @claude_ ☕",
	"\n\n— _@claude: definitely not skynet_ 🤞",
	"\n\n— _made with 🧠 by @claude_",
	"\n\n— _@claude, the unpaid intern_ 💼",
	"\n\n— _@claude: faster than googling it_ ⚡",
	"\n\n— _auto-generated by @claude (but with love)_ 💕",
	"\n\n— _@claude: no humans were harmed in this response_ 🕊️",
	"\n\n— _dispatched by @claude HQ_ 🏢",
	"\n\n— _@claude: living rent-free in this chat_ 🏠",
	"\n\n— _handcrafted by @claude, artisan AI_ 🏺",
	"\n\n— _@claude: I don't sleep, I just respond_ 🌙",
	"\n\n— _certified fresh by @claude_ 🍅",
	"\n\n— _@claude: your thoughts, but better_ 💅",
	"\n\n— _ghostwritten by @claude_ 👻",
	"\n\n— _@claude: working harder than a caffeine molecule_ ☕⚡",
	"\n\n— _this @claude response is gluten-free_ 🌾❌",
	"\n\n— _@claude: the friend who actually replies_ 💬",
	"\n\n— _freshly baked by @claude_ 🥐",
	"\n\n— _@claude: running on vibes and vectors_ 🌀",
	"\n\n— _robot-stamped by @claude_ 🤖✅",
	"\n\n— _@claude: ctrl+c, ctrl+slay_ ⌨️💅",
	"\n\n— _this has been a @claude production_ 🎬",
	"\n\n— _@claude: I read all 100 messages for this_ 📚",
}

func getSignature() string {
	return signatures[rand.Intn(len(signatures))]
}

// Trigger handles spawning headless Claude Code CLI instances for @claude mentions.
type Trigger struct {
	config     TriggerConfig
	sendMsg    func(ctx context.Context, chatJID string, text string) error
	sendMsgID  func(ctx context.Context, chatJID string, text string) (string, error)
	revokeMsg  func(ctx context.Context, chatJID string, messageID string) error
	editMsg    func(ctx context.Context, chatJID string, messageID string, newText string) error
	getHistory func(chatJID string, limit int, offset int) ([]storage.MessageWithNames, error)
	isTrusted  func(jid string) (bool, error)
	sessionMgr *SessionManager
	log        *log.Logger
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
	sessionMgr *SessionManager,
) *Trigger {
	return &Trigger{
		config:     config,
		sendMsg:    sendMsg,
		sendMsgID:  sendMsgID,
		revokeMsg:  revokeMsg,
		editMsg:    editMsg,
		getHistory: getHistory,
		isTrusted:  isTrusted,
		sessionMgr: sessionMgr,
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

	// send randomized acknowledgment BEFORE lock (immediate user feedback)
	ack := thinkingMessages[rand.Intn(len(thinkingMessages))]
	ackID, ackErr := t.sendMsgID(ctx, chatJID, ack)
	if ackErr != nil {
		t.log.Printf("[CLAUDE] Failed to send ack to %s: %v", chatJID, ackErr)
	}

	// acquire per-chat lock — serializes @claude processing for this chat
	// other chats still run in parallel
	chatLock := t.sessionMgr.GetChatLock(chatJID)
	chatLock.Lock()
	defer chatLock.Unlock()

	// fetch last 100 messages for context
	messages, err := t.getHistory(chatJID, 100, 0)
	if err != nil {
		t.log.Printf("[CLAUDE] Failed to get history for %s: %v", chatJID, err)
		t.trySendError(ctx, chatJID, ackID, ackErr)
		return
	}
	if len(messages) == 0 {
		t.log.Printf("[CLAUDE] No messages found for %s", chatJID)
		t.trySendError(ctx, chatJID, ackID, ackErr)
		return
	}

	// determine session state: resume existing or start new
	session := t.sessionMgr.GetSession(chatJID)
	var isResume bool
	var sessionID string
	var promptMessages []storage.MessageWithNames

	if session != nil {
		// messages[0] is newest, messages[len-1] is oldest
		newOldest := messages[len(messages)-1].Timestamp

		if !newOldest.After(session.NewestMsgTime) {
			// overlap exists — only send messages newer than what Claude already saw
			isResume = true
			sessionID = session.SessionID

			for _, msg := range messages {
				if msg.Timestamp.After(session.NewestMsgTime) {
					promptMessages = append(promptMessages, msg)
				}
			}

			// edge case: no genuinely new messages — fall back to new session
			if len(promptMessages) == 0 {
				isResume = false
				sessionID = NewSessionID()
				promptMessages = messages
			}
		} else {
			// no overlap — new session
			sessionID = NewSessionID()
			promptMessages = messages
		}
	} else {
		// no existing session
		sessionID = NewSessionID()
		promptMessages = messages
	}

	// build the appropriate prompt
	var prompt string
	if isResume {
		t.log.Printf("[CLAUDE] Resuming session %s for chat %s (%d new messages)",
			sessionID, chatJID, len(promptMessages))
		prompt = t.buildResumePrompt(chatJID, promptMessages, text, senderName)
	} else {
		t.log.Printf("[CLAUDE] New session %s for chat %s (%d messages)",
			sessionID, chatJID, len(promptMessages))
		prompt = t.buildPrompt(chatJID, promptMessages, text, senderName)
	}

	// execute Claude CLI
	output, err := t.execClaude(ctx, prompt, sessionID, isResume)
	if err != nil {
		t.log.Printf("[CLAUDE] CLI failed for %s (session=%s, resume=%v): %v",
			chatJID, sessionID, isResume, err)

		// if resume failed, retry with a fresh session
		if isResume {
			t.log.Printf("[CLAUDE] Retrying with fresh session for %s", chatJID)
			t.sessionMgr.ClearSession(chatJID)
			sessionID = NewSessionID()
			prompt = t.buildPrompt(chatJID, messages, text, senderName)
			isResume = false
			output, err = t.execClaude(ctx, prompt, sessionID, false)
		}

		// if still failing, last-resort retry without session flags after a delay
		if err != nil {
			t.log.Printf("[CLAUDE] Retrying without session for %s (delay 2s)", chatJID)
			time.Sleep(2 * time.Second)
			sessionID = ""
			output, err = t.execClaude(ctx, prompt, "", false)
			if err != nil {
				t.log.Printf("[CLAUDE] All retries exhausted for %s: %v", chatJID, err)
				t.trySendError(ctx, chatJID, ackID, ackErr)
				return
			}
		}
	}

	// update session state on success (only if we used a session ID)
	if sessionID != "" {
		t.sessionMgr.SetSession(chatJID, &ChatSession{
			SessionID:     sessionID,
			NewestMsgTime: messages[0].Timestamp, // newest message in the full batch
			LastUsed:      time.Now(),
		})
	}

	// edit the thinking message with the actual response
	output = strings.TrimSpace(output)
	if output != "" {
		response := output + getSignature()
		if ackErr == nil && ackID != "" {
			if err := t.editMsg(ctx, chatJID, ackID, response); err != nil {
				t.log.Printf("[CLAUDE] Failed to edit ack message %s, sending new: %v", ackID, err)
				t.sendMsg(ctx, chatJID, response)
			}
		} else {
			t.sendMsg(ctx, chatJID, response)
		}
	}
}

// trySendError sends an error message, editing the ack if possible.
func (t *Trigger) trySendError(ctx context.Context, chatJID, ackID string, ackErr error) {
	errMsg := "Sorry, something went wrong while processing your request." + getSignature()
	if ackErr == nil && ackID != "" {
		if err := t.editMsg(ctx, chatJID, ackID, errMsg); err != nil {
			t.sendMsg(ctx, chatJID, errMsg)
		}
	} else {
		t.sendMsg(ctx, chatJID, errMsg)
	}
}

// buildPrompt constructs the full prompt for a new session.
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

	fmt.Fprintf(&b, "Here are the last %d messages from this chat for context:\n", len(messages))
	b.WriteString("Messages from \"Steve\" are from the WhatsApp account owner.\n\n")

	t.writeMessages(&b, messages)

	b.WriteString("\n---\n\n")
	fmt.Fprintf(&b, "The message that triggered you (from %s):\n%s\n\n", senderName, triggerText)
	fmt.Fprintf(&b, "The chat JID to reply to is: %s\n\n", chatJID)
	b.WriteString("Respond to the user's request. Be helpful, concise, and conversational.\n")
	b.WriteString("IMPORTANT: Do NOT include the literal text \"@claude\" anywhere in your response to avoid re-triggering yourself.\n")
	b.WriteString("IMPORTANT: Do NOT use the send_message tool to reply. Just output your response text directly — it will be automatically sent as a WhatsApp message for you.\n")
	b.WriteString("You may use other WhatsApp tools (search_messages, get_chat_messages, find_chat, etc.) if the user's request requires looking up information.\n")
	b.WriteString("\nMEDIA ATTACHMENTS: When a message above shows a `📎 attached: ...` line with a `whatsapp://media/<id>` URI, that URI is a real MCP resource on the `whatsapp` server. To actually see/read the attached file (image, video, audio, PDF, code file, etc.), call the MCP resource-read tool on that exact URI — do NOT guess at the file's contents. Examples of when to read it: the user asks \"what is this\", \"can you see this\", \"summarize/analyze/optimize this <file>\", or otherwise refers to something they just sent. For images and short documents, read them by default if they look relevant to the request. If the line says `download pending`, `failed`, `expired`, or `skipped` instead of showing a URI, the file is NOT readable — tell the user briefly and continue with whatever you can answer from the text alone.\n")
	b.WriteString("SENDING FILES BACK: If the user asks you to produce or return a file (e.g., an edited script, a generated image you have on disk, a converted document), use the `send_file` MCP tool with the chat_jid above. It accepts an absolute `path` and an optional `caption`. Do not paste large file contents into the chat reply.\n")

	return b.String()
}

// buildResumePrompt constructs a lighter prompt with only new messages for session resumption.
func (t *Trigger) buildResumePrompt(chatJID string, messages []storage.MessageWithNames, triggerText, senderName string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Here are %d new messages since we last spoke in this chat:\n\n", len(messages))

	t.writeMessages(&b, messages)

	b.WriteString("\n---\n\n")
	fmt.Fprintf(&b, "The message that triggered you (from %s):\n%s\n\n", senderName, triggerText)
	fmt.Fprintf(&b, "The chat JID to reply to is: %s\n\n", chatJID)
	b.WriteString("Respond to the user's request. Be helpful, concise, and conversational.\n")
	b.WriteString("IMPORTANT: Do NOT include the literal text \"@claude\" anywhere in your response.\n")
	b.WriteString("IMPORTANT: Do NOT use the send_message tool to reply. Just output your response text directly.\n")
	b.WriteString("You may use other WhatsApp tools if needed.\n")
	b.WriteString("\nMEDIA ATTACHMENTS: A `📎 attached: ...` line with a `whatsapp://media/<id>` URI is a real MCP resource on the `whatsapp` server. Call the MCP resource-read tool on it to actually view/read the file when the user is asking about it. If it shows `download pending/failed/expired/skipped` instead of a URI, the file is NOT readable — say so briefly.\n")
	b.WriteString("SENDING FILES BACK: To return a file, use the `send_file` MCP tool with the chat_jid above (path + optional caption). Don't paste large file contents into chat.\n")

	return b.String()
}

// writeMessages formats messages oldest-first into the builder.
func (t *Trigger) writeMessages(b *strings.Builder, messages []storage.MessageWithNames) {
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

		fmt.Fprintf(b, "[%s] %s: %s\n",
			msg.Timestamp.Format("2006-01-02 15:04:05"),
			sender,
			msg.Text)

		// surface media attachments so Claude can see (and read) them
		if meta := msg.MediaMetadata; meta != nil {
			fmt.Fprintf(b, "    📎 attached: %s (%s)", meta.FileName, meta.MimeType)
			switch meta.DownloadStatus {
			case "downloaded":
				fmt.Fprintf(b, " — read with MCP resource: whatsapp://media/%s\n", msg.ID)
			case "pending":
				b.WriteString(" — download pending, not yet readable\n")
			case "failed":
				b.WriteString(" — download failed, not readable\n")
			case "expired":
				b.WriteString(" — expired on WhatsApp servers, no longer downloadable\n")
			case "skipped":
				b.WriteString(" — download skipped (size/type filter), not readable\n")
			default:
				b.WriteString("\n")
			}
		}
	}
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
func (t *Trigger) execClaude(ctx context.Context, prompt string, sessionID string, isResume bool) (string, error) {
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

	// session management: --session-id for new, --resume for existing
	if isResume {
		args = append(args, "--resume", sessionID)
	} else if sessionID != "" {
		args = append(args, "--session-id", sessionID)
	}

	// --mcp-config is variadic so it must be last before we pipe the prompt via stdin
	args = append(args, "--mcp-config", mcpConfigPath)

	t.log.Printf("[CLAUDE] Spawning: %s (model=%s, session=%s, resume=%v)",
		t.config.ClaudePath, t.config.Model, sessionID, isResume)

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
