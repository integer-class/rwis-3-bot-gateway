package main

import (
	"bytes"
	"encoding/gob"
	"github.com/allegro/bigcache"
	"github.com/pkg/errors"
	types "go.mau.fi/whatsmeow/types"
)

type ChatContext struct {
	SenderId types.JID
	Items    []Content
}

func storeChatContext(cache *bigcache.BigCache, context ChatContext) error {
	var encodedContext bytes.Buffer
	encoder := gob.NewEncoder(&encodedContext)
	err := encoder.Encode(context)
	if err != nil {
		return errors.Wrap(err, "failed to encode chat context")
	}

	err = cache.Set(context.SenderId.String(), encodedContext.Bytes())
	if err != nil {
		return errors.Wrap(err, "failed to store chat context")
	}

	return nil
}

func getChatContext(cache *bigcache.BigCache, senderId types.JID) (ChatContext, error) {
	var decodedContext ChatContext
	encodedContext, err := cache.Get(senderId.String())
	if err != nil {
		return decodedContext, errors.Wrap(err, "failed to get chat context")
	}

	decoder := gob.NewDecoder(bytes.NewReader(encodedContext))
	err = decoder.Decode(&decodedContext)
	if err != nil {
		return decodedContext, errors.Wrap(err, "failed to decode chat context")
	}

	return decodedContext, nil
}
