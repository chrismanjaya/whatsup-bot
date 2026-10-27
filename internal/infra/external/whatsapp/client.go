package whatsapp

import (
	"context"
	"fmt"
	"os"
	"whatsupbot/internal/util"

	qrcode "github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type (
	Client struct {
		whatsmeowClient *whatsmeow.Client
		config          Config
	}
	Config struct {
		StorePath string
	}
)

const (
	clientDialect = "sqlite3"

	clientAddressTemplate = "file:%s?_foreign_keys=on"
)

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	var (
		address string = fmt.Sprintf(clientAddressTemplate, cfg.StorePath)

		databaseLog waLog.Logger = waLog.Stdout("Database", "INFO", true)
		clientLog   waLog.Logger = waLog.Stdout("Client", "INFO", true)
	)

	// set session storage
	container, err := sqlstore.New(ctx, clientDialect, address, databaseLog)
	if err != nil {
		return nil, util.Wrap(err, "open whatsmeow session storage failed")
	}

	// get device
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, util.Wrap(err, "get whatsmeow device failed")
	}

	return &Client{
		whatsmeowClient: whatsmeow.NewClient(deviceStore, clientLog),
		config:          cfg,
	}, nil
}

func (c *Client) Connect(ctx context.Context) error {
	if c.whatsmeowClient.Store.ID == nil {
		qrChan, _ := c.whatsmeowClient.GetQRChannel(ctx)
		if err := c.whatsmeowClient.Connect(); err != nil {
			return util.Wrap(err, "connect whatsmeow client failed")
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan this QR code with WhatsApp:")
				qrcode.GenerateHalfBlock(evt.Code, qrcode.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else if err := c.whatsmeowClient.Connect(); err != nil {
		return util.Wrap(err, "connect whatsmeow client failed")
	}

	if err := c.whatsmeowClient.SendPresence(ctx, types.PresenceAvailable); err != nil {
		util.LogError("set presence failed", err)
	}

	return nil
}

func (c *Client) Client() *whatsmeow.Client {
	return c.whatsmeowClient
}

func (c *Client) Disconnect() {
	c.whatsmeowClient.Disconnect()
}
