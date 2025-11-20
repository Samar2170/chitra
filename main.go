package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Todo model
type Todo struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      int64  `gorm:"index"` // Telegram user ID
	Name        string `gorm:"not null"`
	Details     string
	Priority    string `gorm:"default:'medium'"`
	CreatedAt   time.Time
	Completed   bool `gorm:"default:false"`
	CompletedAt time.Time
}

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("todos.db"), &gorm.Config{})
	if err != nil {
		log.Panic("Failed to connect to database:", err)
	}

	// Auto migrate schema
	err = db.AutoMigrate(&Todo{})
	if err != nil {
		log.Panic("Failed to migrate database:", err)
	}

	log.Println("Database connected and migrated (SQLite: todos.db)")
}

func main() {
	var err error
	args := os.Args
	if len(args) > 1 && (args[1] == "-p" || args[1] == "--prod") {
		currentFile, err := os.Executable()
		if err != nil {
			panic(err)
		}
		ProjectBaseDir := filepath.Dir(currentFile)
		err = godotenv.Load(ProjectBaseDir + "/.env")
	} else {
		err = godotenv.Load()
	}
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	initDB()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized as @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		msg := update.Message
		chatID := msg.Chat.ID
		userID := msg.From.ID

		if !msg.IsCommand() {
			continue
		}

		switch msg.Command() {
		case "start":
			reply := `Welcome to TodoBot! Persistent & Powerful

Commands:
/list - Show your todos
/add Buy groceries, Milk and bread, high - Add new todo
/get 5 - View todo by ID
/done 5 - Mark as completed

Your todos are saved forever!`
			bot.Send(tgbotapi.NewMessage(chatID, reply))

		case "list":
			var todos []Todo
			db.Where("user_id = ? AND completed = ?", userID, false).Order("priority desc, created_at desc").Find(&todos)

			if len(todos) == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, "No pending todos! Use /add to create one."))
				continue
			}

			response := "*Your Active Todos:*\n\n"
			for _, t := range todos {
				emoji := priorityEmoji(t.Priority)
				response += fmt.Sprintf("%s *%d.* %s\n_%s_\n\n", emoji, t.ID, escape(t.Name), escape(t.Details))
			}

			// Also show completed count
			var completedCount int64
			db.Model(&Todo{}).Where("user_id = ? AND completed = ?", userID, true).Count(&completedCount)
			if completedCount > 0 {
				response += fmt.Sprintf("_You have %d completed todo(s)._ Use /done to mark them complete.", completedCount)
			}

			send := tgbotapi.NewMessage(chatID, response)
			send.ParseMode = "Markdown"
			bot.Send(send)

		case "add":
			args := msg.CommandArguments()
			if strings.TrimSpace(args) == "" {
				bot.Send(tgbotapi.NewMessage(chatID, "Usage: /add Name, Details, priority (low/medium/high)"))
				continue
			}

			parts := strings.SplitN(args, ",", 3)
			if len(parts) != 3 {
				bot.Send(tgbotapi.NewMessage(chatID, "Format: Name, Details, priority\nExample: /add Call John, Discuss project timeline, high"))
				continue
			}

			name := strings.TrimSpace(parts[0])
			details := strings.TrimSpace(parts[1])
			priority := strings.ToLower(strings.TrimSpace(parts[2]))

			if name == "" {
				bot.Send(tgbotapi.NewMessage(chatID, "Todo name cannot be empty!"))
				continue
			}

			if !isValidPriority(priority) {
				bot.Send(tgbotapi.NewMessage(chatID, "Priority must be: low, medium, or high"))
				continue
			}

			todo := Todo{
				UserID:    userID,
				Name:      name,
				Details:   details,
				Priority:  priority,
				CreatedAt: time.Now(),
			}

			if err := db.Create(&todo).Error; err != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "Failed to save todo"))
				continue
			}

			reply := fmt.Sprintf("Todo added! (ID: %d)\n\n%s *%s*\n_%s_\nPriority: %s",
				todo.ID, priorityEmoji(priority), escape(name), escape(details), strings.Title(priority))

			send := tgbotapi.NewMessage(chatID, reply)
			send.ParseMode = "Markdown"
			bot.Send(send)

		case "get":
			idStr := msg.CommandArguments()
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil || id == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, "Usage: /get <id>\nExample: /get 5"))
				continue
			}

			var todo Todo
			result := db.Where("id = ? AND user_id = ?", id, userID).First(&todo)
			if result.Error != nil {
				bot.Send(tgbotapi.NewMessage(chatID, "Todo not found or not yours!"))
				continue
			}

			status := "Pending"
			if todo.Completed {
				status = "Completed"
			}

			response := fmt.Sprintf(`*Todo #%d*

*Name:* %s
*Details:* %s
*Priority:* %s
*Status:* %s
*Created:* %s`,
				todo.ID,
				escape(todo.Name),
				escape(todo.Details),
				strings.Title(todo.Priority),
				status,
				todo.CreatedAt.Format("2006-01-02 15:04"),
			)

			if todo.Completed {
				response += fmt.Sprintf("\n*Completed:* %s", todo.CompletedAt.Format("2006-01-02 15:04"))
			}

			send := tgbotapi.NewMessage(chatID, response)
			send.ParseMode = "Markdown"
			bot.Send(send)

		case "done":
			idStr := msg.CommandArguments()
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil || id == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, "Usage: /done <id>"))
				continue
			}

			result := db.Model(&Todo{}).
				Where("id = ? AND user_id = ?", id, userID).
				Updates(map[string]interface{}{
					"Completed":   true,
					"CompletedAt": time.Now(),
				})

			if result.Error != nil || result.RowsAffected == 0 {
				bot.Send(tgbotapi.NewMessage(chatID, "Todo not found or already completed!"))
				continue
			}

			bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Todo #%d marked as done!", id)))
		}
	}
}

// Helpers
func priorityEmoji(p string) string {
	switch p {
	case "high":
		return "High Priority"
	case "medium":
		return "Medium Priority"
	case "low":
		return "Low Priority"
	default:
		return "Medium Priority"
	}
}

func isValidPriority(p string) bool {
	return p == "low" || p == "medium" || p == "high"
}

func escape(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "`", "\\`", "~", "\\~",
	)
	return replacer.Replace(text)
}
