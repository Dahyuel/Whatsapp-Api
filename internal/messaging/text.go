package messaging

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// SendText sends a plain text message directly (bypasses queue).
func SendText(ctx context.Context, client *whatsmeow.Client, jid types.JID, text string) (string, error) {
	msg := &waProto.Message{
		Conversation: proto.String(text),
	}
	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("send text: %w", err)
	}
	return resp.ID, nil
}

// BuildTextMessage constructs a WhatsMeow text message proto.
func BuildTextMessage(text string) *waProto.Message {
	return &waProto.Message{
		Conversation: proto.String(text),
	}
}

// BuildExtendedTextMessage builds a text message with link preview support.
func BuildExtendedTextMessage(text string) *waProto.Message {
	return &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
		},
	}
}

// BuildReplyMessage creates a quoted reply to another message.
func BuildReplyMessage(text, quotedID, quotedJID, quotedText string) *waProto.Message {
	return &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      proto.String(quotedID),
				Participant:   proto.String(quotedJID),
				QuotedMessage: &waProto.Message{Conversation: proto.String(quotedText)},
			},
		},
	}
}

// ParseJID parses a phone number or JID string into a WhatsMeow JID.
func ParseJID(rawJID string) (types.JID, error) {
	return types.ParseJID(rawJID)
}
