package main

import (
	"encoding/gob"
	"github.com/allegro/bigcache"
	"os"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func App() *cli.App {
	return &cli.App{
		Name:        "rwis-bot",
		Description: "A gateway for RWIS-3 System",
		Commands: []*cli.Command{
			{
				Name: "start",
				Action: func(c *cli.Context) error {
					// register ContextItem for gob
					gob.Register(Content{})

					bigCacheConfig := bigcache.Config{
						Shards:             1024,
						MaxEntriesInWindow: 1000 * 10 * 60,
						MaxEntrySize:       500,
						Verbose:            true,
						HardMaxCacheSize:   1024,
						OnRemove:           nil,
						OnRemoveWithReason: nil,
					}
					cache, err := bigcache.NewBigCache(bigCacheConfig)
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to create cache")
					}

					dbLog := waLog.Stdout("Database", "DEBUG", true)
					container, err := sqlstore.New("sqlite3", "file:store.db?_foreign_keys=on", dbLog)
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to create db store container")
					}

					// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
					deviceStore, err := container.GetFirstDevice()
					if err != nil {
						log.Fatal().Err(err).Msg("Failed to get device")
					}

					clientLog := waLog.Stdout("Client", "DEBUG", true)
					bot := &Bot{
						client: whatsmeow.NewClient(deviceStore, clientLog),
						cache:  cache,
					}
					bot.RegisterHandlers()

					if err := bot.Start(); err != nil {
						log.Fatal().Err(err).Msg("Failed to start bot")
					}

					return nil
				},
			},
		},
	}
}

func main() {
	if err := App().Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("Failed to run app")
	}
}
