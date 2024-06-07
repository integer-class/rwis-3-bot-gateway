package main

import (
	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
	"net/http"
	"os"
)

type Server struct {
	bot    *Bot
	server *chi.Mux
	token  string
}

func (s *Server) Start() error {
	s.server = chi.NewRouter()
	s.server.Post("/api/v1/broadcast", s.handleBroadcast)

	s.token = os.Getenv("BROADCAST_TOKEN")
	if s.token == "" {
		return errors.New("BROADCAST_TOKEN is not set")
	}

	return http.ListenAndServe(":8080", s.server)
}

func (s *Server) handleBroadcast(res http.ResponseWriter, req *http.Request) {
	// make sure the request has the token to prevent abuse
	if req.Header.Get("Authorization") != "Bearer "+s.token {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Debug().Msg("Received broadcast request")

	// get the message from the request body
	message := req.FormValue("message")
	// list of numbers to send the message to
	number := req.FormValue("number")

	// debug
	log.Debug().Msg("Query: " + req.URL.RawQuery)

	log.Debug().Msgf("Broadcasting message to number: %s", number)

	// send the message to the numbers
	_, err := s.bot.client.SendMessage(req.Context(), types.JID{User: number, Server: "s.whatsapp.net"}, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(message),
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to send message")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Debug().Msg("Broadcast message sent")
	res.WriteHeader(http.StatusOK)
}
