package channels

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// GetInfo returns metadata for a newsletter (channel) JID.
func GetInfo(ctx context.Context, client *whatsmeow.Client, jid types.JID) (*types.NewsletterMetadata, error) {
	info, err := client.GetNewsletterInfo(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("get channel info: %w", err)
	}
	return info, nil
}

// List returns all channels the bot has joined.
func List(ctx context.Context, client *whatsmeow.Client) ([]*types.NewsletterMetadata, error) {
	return client.GetSubscribedNewsletters(ctx)
}

// SendText sends a text message to a channel.
func SendText(ctx context.Context, client *whatsmeow.Client, jid types.JID, text string) (string, error) {
	msg := &waProto.Message{
		Conversation: proto.String(text),
	}
	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return "", fmt.Errorf("send channel message: %w", err)
	}
	return resp.ID, nil
}
