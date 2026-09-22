package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	qrcode "github.com/mdp/qrterminal/v3"
)

func main() {
	ctx := context.Background()

	dbLog := waLog.Stdout("Database", "INFO", true)
	// session data will be stored in whatsmeow.db in this folder
	container, err := sqlstore.New(ctx, "sqlite3", "file:whatsmeow.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// handler for incoming messages
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			text := v.Message.GetConversation()
			if text == "" && v.Message.GetExtendedTextMessage() != nil {
				text = v.Message.GetExtendedTextMessage().GetText()
			}
			if text == "" {
				return
			}

			// avoid infinite loop: skip messages that are the bot's own echoed replies
			if v.Info.IsFromMe && strings.HasPrefix(text, "Echo: ") {
				return
			}

			fmt.Printf("Received from %s: %s\n", v.Info.Sender.String(), text)

			reply := &waE2E.Message{
				Conversation: proto.String("Echo: " + text),
			}
			_, err := client.SendMessage(context.Background(), v.Info.Chat, reply)
			if err != nil {
				fmt.Println("Error sending reply:", err)
			}
		}
	})

	if client.Store.ID == nil {
		// not logged in yet, need to pair via QR code
		qrChan, _ := client.GetQRChannel(ctx)
		err = client.Connect()
		if err != nil {
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
		// already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Bot is running. Press Ctrl+C to exit.")

	// wait for interrupt signal to gracefully shut down
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	client.Disconnect()
}
