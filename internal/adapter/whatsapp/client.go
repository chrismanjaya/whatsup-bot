package whatsapp

import (
	"context"
	"fmt"
	"os"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	qrcode "github.com/mdp/qrterminal/v3"

	"whatsup-bot/internal/utils"
)

// NewClient opens whatsmeow's session store at storePath and builds a client
// on its first device. It does not connect; call Connect for that.
func NewClient(ctx context.Context, storePath string) (*whatsmeow.Client, error) {
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+storePath+"?_foreign_keys=on", waLog.Stdout("Database", "INFO", true))
	if err != nil {
		return nil, utils.Wrap(err, "open whatsmeow store")
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, utils.Wrap(err, "get whatsmeow device")
	}
	return whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true)), nil
}

// Connect connects the client, walking through QR pairing first when the
// device isn't linked yet, then marks the bot as available.
func Connect(ctx context.Context, client *whatsmeow.Client) error {
	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			return utils.Wrap(err, "connect whatsmeow client")
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan this QR code with WhatsApp:")
				qrcode.GenerateHalfBlock(evt.Code, qrcode.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else if err := client.Connect(); err != nil {
		return utils.Wrap(err, "connect whatsmeow client")
	}

	if err := client.SendPresence(ctx, types.PresenceAvailable); err != nil {
		utils.LogError("set presence failed", err)
	}
	return nil
}
