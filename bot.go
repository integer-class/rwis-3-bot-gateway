package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type Bot struct {
	client *whatsmeow.Client
}

func (b *Bot) RegisterHandlers() {
	b.client.AddEventHandler(b.pingHandler)
}

func (b *Bot) Start() error {
	if b.client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := b.client.GetQRChannel(context.Background())
		err := b.client.Connect()
		if err != nil {
			return errors.Wrap(err, "failed to connect")
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				qrterminal.Generate(evt.Code, qrterminal.L, os.Stdout)
			} else {
				log.Debug().Msg("Login event: " + evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err := b.client.Connect()
		if err != nil {
			return errors.Wrap(err, "failed to connect")
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, os.Interrupt, syscall.SIGTERM)
	<-exitSig

	b.client.Disconnect()
	return nil
}

func (b *Bot) pingHandler(evt interface{}) {
	startTime := time.Now()
	switch v := evt.(type) {
	case *events.Message:
		msg := v.Message.GetConversation()
		// if the message is empty it means that the message is an extended text message
		if msg == "" {
			msg = *v.Message.GetExtendedTextMessage().Text
		}

		if msg == "ping" {
			fmt.Println("Sending pong")
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			v.Info.Sender.Device = 0
			message, err := b.client.SendMessage(ctx, v.Info.Sender, &waProto.Message{
				ExtendedTextMessage: &waProto.ExtendedTextMessage{
					Text: proto.String("Pong! Response Time: " + strconv.FormatInt(time.Since(startTime).Nanoseconds(), 10) + "ns"),
				},
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to send message")
			} else {
				log.Debug().Msgf("Sent message %+v", message)
			}
		}

		//msg := *v.Message.GetExtendedTextMessage().Text
		//if msg == "ping" {
		//	fmt.Println("Sending pong")
		//	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		//	defer cancel()
		//	fmt.Println(v.Info.Sender.Device)
		//	message, err := b.client.SendMessage(ctx, v.Info.Sender, &waProto.Message{
		//		ExtendedTextMessage: &waProto.ExtendedTextMessage{
		//			Text: proto.String("pong"),
		//		},
		//	})
		//	if err != nil {
		//		log.Error().Err(err).Msg("Failed to send message")
		//	} else {
		//		log.Debug().Msgf("Sent message %+v", message)
		//	}
		//}
	}
}
