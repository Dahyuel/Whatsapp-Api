package chats

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ListChats returns the list of recent chats.
func ListChats(ctx context.Context, client *whatsmeow.Client) (interface{}, error) {
	return nil, fmt.Errorf("list chats: unsupported in this whatsmeow version")
}

// GetMessages returns recent messages for a chat.
func GetMessages(ctx context.Context, client *whatsmeow.Client, jid types.JID, count int) (interface{}, error) {
	return nil, fmt.Errorf("get messages: unsupported in this whatsmeow version")
}

// MarkRead marks all messages in a chat as read.
func MarkRead(ctx context.Context, client *whatsmeow.Client, jid types.JID, msgIDs []types.MessageID) error {
	return client.MarkRead(ctx, msgIDs, time.Now(), jid, types.EmptyJID)
}

// SetMuted mutes or unmutes a chat.
func SetMuted(ctx context.Context, client *whatsmeow.Client, jid types.JID, muted bool) error {
	return fmt.Errorf("set muted: unsupported in this whatsmeow version")
}

// SetPinned pins or unpins a chat.
func SetPinned(ctx context.Context, client *whatsmeow.Client, jid types.JID, pinned bool) error {
	return fmt.Errorf("set pinned: unsupported in this whatsmeow version")
}

// SetArchived archives or unarchives a chat.
func SetArchived(ctx context.Context, client *whatsmeow.Client, jid types.JID, archived bool) error {
	return fmt.Errorf("set archived: unsupported in this whatsmeow version")
}
