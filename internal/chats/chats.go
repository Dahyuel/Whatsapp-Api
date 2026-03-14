package chats

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ListChats returns the list of recent chats.
func ListChats(ctx context.Context, client *whatsmeow.Client) (interface{}, error) {
	chats, err := client.Store.Chats.GetAllChats()
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	return chats, nil
}

// GetMessages returns recent messages for a chat.
func GetMessages(ctx context.Context, client *whatsmeow.Client, jid types.JID, count int) (interface{}, error) {
	msgs, err := client.Store.Messages.GetMessagesBefore(jid, nil, count)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	return msgs, nil
}

// MarkRead marks all messages in a chat as read.
func MarkRead(ctx context.Context, client *whatsmeow.Client, jid types.JID, msgIDs []types.MessageID) error {
	now := client.Store.ID
	if now == nil {
		return fmt.Errorf("client not connected")
	}
	return client.MarkRead(msgIDs, types.EmptyJID, jid, *now)
}

// SetMuted mutes or unmutes a chat.
func SetMuted(ctx context.Context, client *whatsmeow.Client, jid types.JID, muted bool) error {
	return client.SetMuted(jid, muted, 0)
}

// SetPinned pins or unpins a chat.
func SetPinned(ctx context.Context, client *whatsmeow.Client, jid types.JID, pinned bool) error {
	return client.SetPinned(jid, pinned)
}

// SetArchived archives or unarchives a chat.
func SetArchived(ctx context.Context, client *whatsmeow.Client, jid types.JID, archived bool) error {
	return client.SetArchived(jid, archived)
}
