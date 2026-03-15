package chats

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"whatsapp-api/internal/db"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ListChats returns the list of recent chats.
func ListChats(ctx context.Context, database *sql.DB, sessionID string) ([]*db.ChatRow, error) {
	return db.ListChats(database, sessionID)
}

// GetMessages returns recent messages for a chat.
func GetMessages(ctx context.Context, database *sql.DB, sessionID string, jid types.JID, count int) ([]*db.ChatMessageRow, error) {
	return db.GetChatMessages(database, sessionID, jid.String(), count)
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
