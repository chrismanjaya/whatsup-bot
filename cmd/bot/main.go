package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
	qrcode "github.com/mdp/qrterminal/v3"
	"google.golang.org/genai"
	"google.golang.org/protobuf/proto"

	"whatsup-bot/internal/adapter/gemini"
	"whatsup-bot/internal/adapter/sqlite"
	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/usecase"
	"whatsup-bot/internal/utils"
)

func main() {
	ctx := context.Background()

	utils.SetupLogger()

	appDB, err := sqlite.Open("whatsup.db")
	if err != nil {
		utils.Fatal("open sqlite db failed", err)
	}

	userRepo := sqlite.NewUserRepo(appDB)
	txRepo := sqlite.NewTransactionRepo(appDB)
	groupRepo := sqlite.NewGroupRepo(appDB)

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		utils.Fatal("create gemini client failed", err)
	}
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-3.6-flash"
	}
	parser := gemini.NewParser(geminiClient, geminiModel)

	registerUser := usecase.NewRegisterUserUseCase(userRepo)
	recordTx := usecase.NewRecordTransactionUseCase(userRepo, txRepo, parser)
	amendTx := usecase.NewAmendTransactionUseCase(userRepo, txRepo, parser)
	ensureGroup := usecase.NewEnsureGroupUseCase(groupRepo)
	ensureMembership := usecase.NewEnsureGroupMembershipUseCase(userRepo, groupRepo)
	computeSplit := usecase.NewComputeSplitUseCase(groupRepo, txRepo)

	dbLog := waLog.Stdout("Database", "INFO", true)
	waContainer, err := sqlstore.New(ctx, "sqlite3", "file:whatsmeow.db?_foreign_keys=on", dbLog)
	if err != nil {
		utils.Fatal("open whatsmeow store failed", err)
	}
	deviceStore, err := waContainer.GetFirstDevice(ctx)
	if err != nil {
		utils.Fatal("get whatsmeow device failed", err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(func(evt interface{}) {
		v, ok := evt.(*events.Message)
		if !ok {
			return
		}
		if v.Info.IsFromMe {
			return
		}

		text := v.Message.GetConversation()
		stanzaID := ""
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			if text == "" {
				text = ext.GetText()
			}
			if ci := ext.GetContextInfo(); ci != nil {
				stanzaID = ci.GetStanzaID()
			}
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}

		senderJID := v.Info.Sender.ToNonAD().String()
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
				utils.LogError("ensure group failed", err, "group_jid", chatJID.String())
				return
			}
			groupID = gid

			if err := ensureMembership.Execute(ctx, senderJID, groupID); err != nil {
				utils.LogError("ensure membership failed", err, "sender", senderJID, "group_id", groupID)
			}
		}

		if err := client.SendChatPresence(ctx, chatJID, types.ChatPresenceComposing, types.ChatPresenceMediaText); err != nil {
			utils.LogError("send typing presence failed", err, "chat_jid", chatJID.String())
		}

		reply, tx := handleMessage(ctx, text, stanzaID, senderJID, isGroup, groupID, registerUser, recordTx, amendTx, computeSplit)

		if err := client.SendChatPresence(ctx, chatJID, types.ChatPresencePaused, types.ChatPresenceMediaText); err != nil {
			utils.LogError("clear typing presence failed", err, "chat_jid", chatJID.String())
		}

		if reply == "" {
			return
		}

		msg := &waE2E.Message{Conversation: proto.String(reply)}
		resp, err := client.SendMessage(ctx, chatJID, msg)
		if err != nil {
			utils.LogError("send reply failed", err, "chat_jid", chatJID.String())
			return
		}

		if tx != nil {
			if err := txRepo.SetWAMessageID(ctx, tx.ID, resp.ID); err != nil {
				utils.LogError("set wa message id failed", err, "tx_id", tx.ID)
			}
		}
	})

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			utils.Fatal("connect whatsmeow client failed", err)
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
			utils.Fatal("connect whatsmeow client failed", err)
		}
	}

	if err := client.SendPresence(ctx, types.PresenceAvailable); err != nil {
		utils.LogError("set presence failed", err)
	}

	fmt.Println("Bot is running. Press Ctrl+C to exit.")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	client.Disconnect()
}

func handleMessage(
	ctx context.Context,
	text, stanzaID, senderJID string,
	isGroup bool,
	groupID int64,
	registerUser *usecase.RegisterUserUseCase,
	recordTx *usecase.RecordTransactionUseCase,
	amendTx *usecase.AmendTransactionUseCase,
	computeSplit *usecase.ComputeSplitUseCase,
) (string, *domain.Transaction) {
	slog.Info("message received", "sender", senderJID, "is_group", isGroup, "group_id", groupID, "text", text, "stanza_id", stanzaID)

	if strings.HasPrefix(text, "*EXPENSE*") || strings.HasPrefix(text, "*INCOME*") || strings.HasPrefix(text, "*DELETED*") || strings.HasPrefix(text, "Welcome") || strings.HasPrefix(text, "You're already registered") {
		return "", nil
	}

	if stanzaID != "" {
		reply, tx, handled, err := amendTx.Execute(ctx, senderJID, stanzaID, text)
		if err != nil {
			utils.LogError("amend transaction failed", err, "sender", senderJID)
			return replyForError(err), nil
		}
		if handled {
			slog.Info("transaction amended", "sender", senderJID, "stanza_id", stanzaID)
			return reply, tx
		}
	}

	if strings.HasPrefix(text, "register ") {
		parts := strings.Fields(text)
		if len(parts) < 3 {
			return "Usage: register <name> <email>", nil
		}
		reply, err := registerUser.Execute(ctx, senderJID, parts[1], parts[2])
		if err != nil {
			utils.LogError("register failed", err, "sender", senderJID)
			return replyForError(err), nil
		}
		slog.Info("user registered", "sender", senderJID, "name", parts[1])
		return reply, nil
	}

	if text == "split" && isGroup {
		reply, err := computeSplit.Execute(ctx, groupID)
		if err != nil {
			utils.LogError("split failed", err, "group_id", groupID)
			return replyForError(err), nil
		}
		slog.Info("split computed", "group_id", groupID)
		return reply, nil
	}

	reply, tx, err := recordTx.Execute(ctx, senderJID, text, isGroup, groupID)
	if err != nil {
		utils.LogError("record transaction failed", err, "sender", senderJID)
		return replyForError(err), nil
	}
	slog.Info("transaction processed", "sender", senderJID, "is_group", isGroup, "recorded", tx != nil)
	return reply, tx
}

// replyForError maps a usecase's classified error to the message shown to
// the user, keeping that mapping in one place instead of per call site. The
// numeric error code is appended so a user can report it without exposing
// any internal detail — a developer can look up what it means from there.
func replyForError(err error) string {
	msg := "Something went wrong. Please try again."
	switch {
	case errors.Is(err, constant.ErrAlreadyExists):
		msg = "You're already registered."
	case errors.Is(err, constant.ErrNotFound):
		msg = "Please register first: register <name> <email>"
	case errors.Is(err, constant.ErrInvalidRequest):
		msg = "I can only help track income and expenses. Try: \"spent 50k on lunch\""
	case errors.Is(err, constant.ErrServiceUnavailable):
		msg = "The service is a bit busy right now, please try sending that again in a moment."
	}

	code := constant.ErrInternal.Code
	if c, ok := utils.CodeOf(err); ok {
		code = c.Code
	}
	return fmt.Sprintf("%s (Error code: %d)", msg, code)
}
