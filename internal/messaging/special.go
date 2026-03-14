package messaging

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// BuildLocationMessage builds a location sharing message.
func BuildLocationMessage(lat, lon float64, name, address string) *waProto.Message {
	return &waProto.Message{
		LocationMessage: &waProto.LocationMessage{
			DegreesLatitude:  proto.Float64(lat),
			DegreesLongitude: proto.Float64(lon),
			Name:             proto.String(name),
			Address:          proto.String(address),
		},
	}
}

// BuildContactMessage builds a vCard contact message.
func BuildContactMessage(displayName, vcard string) *waProto.Message {
	return &waProto.Message{
		ContactMessage: &waProto.ContactMessage{
			DisplayName: proto.String(displayName),
			Vcard:       proto.String(vcard),
		},
	}
}

// BuildReactionMessage builds a reaction to a message.
func BuildReactionMessage(targetMsgID, targetSenderJID, reaction string) *waProto.Message {
	return &waProto.Message{
		ReactionMessage: &waProto.ReactionMessage{
			Key: &waProto.MessageKey{
				ID:          proto.String(targetMsgID),
				FromMe:      proto.Bool(false),
				RemoteJID:   proto.String(targetSenderJID),
			},
			Text:              proto.String(reaction),
			SenderTimestampMS: proto.Int64(0),
		},
	}
}

// BuildPollMessage builds a poll message.
func BuildPollMessage(question string, options []string, selectableCount uint32) *waProto.Message {
	opts := make([]*waProto.PollCreationMessage_Option, 0, len(options))
	for _, o := range options {
		opts = append(opts, &waProto.PollCreationMessage_Option{
			OptionName: proto.String(o),
		})
	}
	return &waProto.Message{
		PollCreationMessage: &waProto.PollCreationMessage{
			Name:                  proto.String(question),
			Options:               opts,
			SelectableOptionsCount: proto.Uint32(selectableCount),
		},
	}
}

// DeleteMessage revokes/deletes a sent message.
func DeleteMessage(ctx context.Context, client *whatsmeow.Client, chatJID types.JID, msgID string, fromMe bool) error {
	buildRevoke := client.BuildRevoke(chatJID, types.EmptyJID, msgID)
	_, err := client.SendMessage(ctx, chatJID, buildRevoke)
	if err != nil {
		return fmt.Errorf("delete message: %w", err)
	}
	return nil
}

// EditMessage edits a previously sent text message.
func EditMessage(ctx context.Context, client *whatsmeow.Client, chatJID types.JID, msgID, newText string) error {
	_, err := client.SendMessage(ctx, chatJID, client.BuildEdit(chatJID, msgID, &waProto.Message{
		Conversation: proto.String(newText),
	}))
	return err
}
