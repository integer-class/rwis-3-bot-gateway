package main

import (
	"context"
	"fmt"
	"github.com/allegro/bigcache"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	cache  *bigcache.BigCache
	db     *pgxpool.Pool
}

func (b *Bot) RegisterHandlers() {
	b.client.AddEventHandler(b.messageHandler)
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

// pingHandler is a simple handler that responds to a ping with a pong along with the response time
func (b *Bot) handlePingEvent(evt *events.Message) {
	startTime := time.Now()
	fmt.Println("Sending pong")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	evt.Info.Sender.Device = 0
	message, err := b.client.SendMessage(ctx, evt.Info.Sender, &waProto.Message{
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

func (b *Bot) handleGeminiEvent(sender string, msg string, evt *events.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	evt.Info.Sender.Device = 0

	// get previous chat contexts from cache
	chatContext, err := getChatContext(b.cache, evt.Info.Sender)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get chat context")
	}

	// fetch answer from gemini
	question := fmt.Sprintf(`{ "sender": "%s", "chat": "%s" }`, evt.Info.Sender, msg)
	geminiAnswer, err := fetchGeminiResponse(question, chatContext.Items)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch response from gemini")
		geminiAnswer = "Maaf, saya tidak bisa membantu Anda saat ini."
	}
	if geminiAnswer == "" {
		geminiAnswer = "Maaf, saya tidak bisa membantu Anda saat ini."
	}

	// try parsing the answer to see if it's a command
	var reply string
	err, output := parseGeminiAnswer(geminiAnswer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse gemini answer. Falling back to default message.")
		reply = geminiAnswer
	} else {
		// handle different type of output accordingly
		switch output["type"] {
		case "personal_data_request":
			log.Debug().Msgf("Handling personal data request from %s", sender)
			err, reply = handlePersonalDataRequest(ctx, b.db, sender, output["include"].(string))
			if err != nil {
				log.Error().Err(err).Msg("Failed to handle personal data request")
				return
			}
		case "issue_report":
			log.Debug().Msgf("Handling issue report from %s", sender)
			meta := output["meta"].(map[string]interface{})
			err, reply = handleIssueReport(ctx, b.db, sender, output["value"].(string), meta["title"].(string), meta["description"].(string))
			if err != nil {
				log.Error().Err(err).Msg("Failed to handle issue report")
				return
			}
		case "chat":
			reply = output["value"].(string)
		default:
			reply = "Maaf, saya tidak bisa membantu Anda saat ini."
		}
	}

	log.Debug().Msgf("Sending reply to %s: %s", sender, reply)
	message, err := b.client.SendMessage(ctx, evt.Info.Sender, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(reply),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message")
	} else {
		log.Debug().Msgf("Sent message %+v", message)
	}

	// store to chat context unique to each user
	err = storeChatContext(b.cache, ChatContext{
		SenderId: evt.Info.Sender,
		Items: []Content{
			{Role: "user", Parts: []Part{{Text: msg}}},
			{Role: "model", Parts: []Part{{Text: geminiAnswer}}},
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to store chat context")
	}
}

func (b *Bot) messageHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// ignore if it's from a group
		if v.Info.IsGroup {
			return
		}

		// whitelist the sender
		senderNumber := strings.Split(v.Info.Sender.String(), "@")[0]
		// if it's coming from whatsapp web, the number will be different, it will have a trailing code separated
		// by a colon, so we need to split it again
		senderNumber = strings.Split(senderNumber, ":")[0]

		msg := extractMessage(v)
		if msg == "ping" {
			b.handlePingEvent(v)
			return
		}
		// handle the rest of the messages using gemini
		b.handleGeminiEvent(senderNumber, msg, v)
		break
	}
}

// extractMessage is used to handle if the message is a conversation or an extended text message
func extractMessage(v *events.Message) string {
	msg := v.Message.GetConversation()
	// if the message is empty it means that the message is an extended text message
	if msg == "" {
		msg = v.Message.GetExtendedTextMessage().GetText()
	}
	return msg
}
