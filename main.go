package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/genai"
)

type GrammarBot struct {
	bot   *tgbotapi.BotAPI
	genAI *genai.Client
	ctx   context.Context
}

func NewGrammarBot(telegramToken, geminiAPIKey string) (*GrammarBot, error) {
	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// Initialize Gemini AI client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  geminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return &GrammarBot{
		bot:   bot,
		genAI: client,
		ctx:   ctx,
	}, nil
}

func (gb *GrammarBot) checkGrammar(text string) (string, error) {
	result, err := gb.genAI.Models.GenerateContent(
		gb.ctx,
		"gemini-2.5-flash-preview-05-20",
		genai.Text(fmt.Sprintf(`System:
You are a world-class English language assistant specializing in grammar and vocabulary correction for Telegram messages using MarkdownV2. When given a user’s sentence, you must:

1. Identify all grammar, spelling, punctuation or word-choice mistakes.  
2. Escape every special MarkdownV2 character (_ * [ ] ( ) ~  > # + - = | { } . !) by prefixing it with a backslash.  
3. Wrap each original mistake in ~strikethrough~ and each correction in **bold**, using valid MarkdownV2 syntax.  
4. Preserve the original meaning, tone and style.  
5. Return exactly the single corrected sentence with those inline edits—no explanations, comments or extra text.

User:
%s`, text)),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return result.Text(), nil
}

func (gb *GrammarBot) handleMessage(message *tgbotapi.Message) {
	// Skip if message is empty or is a command
	if message.Text == "" || strings.HasPrefix(message.Text, "/") {
		return
	}

	// Send "typing" action to show bot is processing
	typingAction := tgbotapi.NewChatAction(message.Chat.ID, tgbotapi.ChatTyping)
	gb.bot.Send(typingAction)

	// Check grammar using Gemini AI
	correctedText, err := gb.checkGrammar(message.Text)
	if err != nil {
		log.Printf("Error checking grammar: %v", err)

		errorMsg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, I encountered an error while checking your grammar. Please try again later.")
		errorMsg.ReplyToMessageID = message.MessageID
		gb.bot.Send(errorMsg)
		return
	}

	// Prepare response message
	responseText := fmt.Sprintf("📝 Grammar check for your message:\n\n%s", correctedText)

	msg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	msg.ReplyToMessageID = message.MessageID
	msg.ParseMode = "MarkdownV2"

	// Send the corrected text
	if _, err := gb.bot.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (gb *GrammarBot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		welcomeText := `👋 Welcome to Grammar Check Bot!

Send me any text message and I'll check it for grammar, spelling, and punctuation errors.

I'll show corrections with:
- ~strikethrough~ for original mistakes
- **bold** for corrections

Commands:
/start - Show this welcome message
/help - Show help information`

		msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
		msg.ParseMode = "MarkdownV2"
		gb.bot.Send(msg)

	case "help":
		helpText := `🔍 How to use Grammar Check Bot:

1. Simply send me any text message
2. I'll analyze it for grammar, spelling, and punctuation errors
3. You'll receive a corrected version with highlighted changes

📝 Example:
Your text: "I goes to store yesterday"
My response: "I ~goes~ **went** to ~store~ **the store** yesterday"

💡 This helps you verify that your message conveys what you intended before sending it elsewhere!`

		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		msg.ParseMode = "MarkdownV2"
		gb.bot.Send(msg)

	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
		gb.bot.Send(msg)
	}
}

func (gb *GrammarBot) Start() error {
	log.Printf("Bot authorized on account %s", gb.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := gb.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Handle commands
		if update.Message.IsCommand() {
			gb.handleCommand(update.Message)
		} else {
			// Handle regular text messages
			gb.handleMessage(update.Message)
		}
	}

	return nil
}

func main() {
	// Get tokens from environment variables
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")

	if telegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}
	if geminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}

	// Create and start the bot
	bot, err := NewGrammarBot(telegramToken, geminiAPIKey)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting Grammar Check Bot...")
	if err := bot.Start(); err != nil {
		log.Fatal(err)
	}
}
