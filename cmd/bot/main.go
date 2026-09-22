package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
	qrcode "github.com/mdp/qrterminal/v3"
	"google.golang.org/genai"
	"google.golang.org/protobuf/proto"

	"whatsup-bot/internal/adapter/gemini"
	"whatsup-bot/internal/adapter/sqlite"
	"whatsup-bot/internal/usecase"
)

func main() {
	ctx := context.Background()

	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(logHandler))

	appDB, err := sqlite.Open("whatsup.db")
	if err != nil {
		panic(err)
	}

	userRepo := sqlite.NewUserRepo(appDB)
	txRepo := sqlite.NewTransactionRepo(appDB)
	groupRepo := sqlite.NewGroupRepo(appDB)

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		panic(err)
	}
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-3.6-flash"
	}
	parser := gemini.NewParser(geminiClient, geminiModel)

	registerUser := usecase.NewRegisterUserUseCase(userRepo)
	recordTx := usecase.NewRecordTransactionUseCase(userRepo, txRepo, parser)
	ensureGroup := usecase.NewEnsureGroupUseCase(groupRepo)
	ensureMembership := usecase.NewEnsureGroupMembershipUseCase(userRepo, groupRepo)
	computeSplit := usecase.NewComputeSplitUseCase(groupRepo, txRepo)

	dbLog := waLog.Stdout("Database", "INFO", true)
	waContainer, err := sqlstore.New(ctx, "sqlite3", "file:whatsmeow.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	deviceStore, err := waContainer.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(func(evt interface{}) {
		v, ok := evt.(*events.Message)
		if !ok {
			return
		}

		text := v.Message.GetConversation()
		if text == "" && v.Message.GetExtendedTextMessage() != nil {
			text = v.Message.GetExtendedTextMessage().GetText()
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}

		senderJID := v.Info.Sender.String()
		chatJID := v.Info.Chat
		isGroup := strings.HasSuffix(chatJID.String(), "@g.us")

		var groupID int64
		if isGroup {
			groupName := chatJID.String()
			if info, err := client.GetGroupInfo(ctx, chatJID); err == nil {
				groupName = info.Name
			}
			gid, err := ensureGroup.Execute(ctx, chatJID.String(), groupName)
			if err != nil {
				fmt.Println("ensure group failed:", err)
				return
			}
			groupID = gid

			if err := ensureMembership.Execute(ctx, senderJID, groupID); err != nil {
				fmt.Println("ensure membership failed:", err)
			}
		}

		reply := handleMessage(ctx, text, senderJID, isGroup, groupID, registerUser, recordTx, computeSplit)
		if reply == "" {
			return
		}

		msg := &waE2E.Message{Conversation: proto.String(reply)}
		if _, err := client.SendMessage(ctx, chatJID, msg); err != nil {
			fmt.Println("send reply failed:", err)
		}
	})

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan this QR code with WhatsApp:")
				qrcode.GenerateHalfBlock(evt.Code, qrcode.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		if err := client.Connect(); err != nil {
			panic(err)
		}
	}

	fmt.Println("Bot is running. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	client.Disconnect()
}

func handleMessage(
	ctx context.Context,
	text, senderJID string,
	isGroup bool,
	groupID int64,
	registerUser *usecase.RegisterUserUseCase,
	recordTx *usecase.RecordTransactionUseCase,
	computeSplit *usecase.ComputeSplitUseCase,
) string {
	slog.Info("message received", "sender", senderJID, "is_group", isGroup, "group_id", groupID, "text", text)

	if strings.HasPrefix(text, "Recorded:") || strings.HasPrefix(text, "Welcome") {
		return ""
	}

	if strings.HasPrefix(text, "register ") {
		parts := strings.Fields(text)
		if len(parts) < 3 {
			return "Usage: register <name> <email>"
		}
		reply, err := registerUser.Execute(ctx, senderJID, parts[1], parts[2])
		if err != nil {
			slog.Error("register failed", "sender", senderJID, "error", err)
			return "Something went wrong registering you. Try again."
		}
		slog.Info("user registered", "sender", senderJID, "name", parts[1])
		return reply
	}

	if text == "split" && isGroup {
		reply, err := computeSplit.Execute(ctx, groupID)
		if err != nil {
			slog.Error("split failed", "group_id", groupID, "error", err)
			return "Couldn't compute the split right now."
		}
		slog.Info("split computed", "group_id", groupID)
		return reply
	}

	reply, recorded, err := recordTx.Execute(ctx, senderJID, text, isGroup, groupID)
	if err != nil {
		slog.Error("record transaction failed", "sender", senderJID, "error", err)
		return "Something went wrong recording that."
	}
	slog.Info("transaction processed", "sender", senderJID, "is_group", isGroup, "recorded", recorded)
	return reply
}
