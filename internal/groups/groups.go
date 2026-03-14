package groups

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// Create creates a new WhatsApp group.
func Create(ctx context.Context, client *whatsmeow.Client, name string, participants []string) (string, error) {
	jids := make([]types.JID, 0, len(participants))
	for _, p := range participants {
		jid, err := types.ParseJID(p)
		if err != nil {
			return "", fmt.Errorf("invalid participant JID %q: %w", p, err)
		}
		jids = append(jids, jid)
	}
	req := whatsmeow.ReqCreateGroup{
		Name:         name,
		Participants: jids,
	}
	info, err := client.CreateGroup(ctx, req)
	if err != nil {
		return "", fmt.Errorf("create group: %w", err)
	}
	return info.JID.String(), nil
}

// GetInfo retrieves group metadata.
func GetInfo(ctx context.Context, client *whatsmeow.Client, groupJID types.JID) (*types.GroupInfo, error) {
	info, err := client.GetGroupInfo(ctx, groupJID)
	if err != nil {
		return nil, fmt.Errorf("get group info: %w", err)
	}
	return info, nil
}

// UpdateParticipants adds or removes participants in a group.
func UpdateParticipants(ctx context.Context, client *whatsmeow.Client, groupJID types.JID, participants []string, action whatsmeow.ParticipantChange) error {
	jids := make([]types.JID, 0, len(participants))
	for _, p := range participants {
		jid, err := types.ParseJID(p)
		if err != nil {
			return fmt.Errorf("invalid JID %q: %w", p, err)
		}
		jids = append(jids, jid)
	}
	_, err := client.UpdateGroupParticipants(ctx, groupJID, jids, action)
	return err
}

// SetDescription updates a group's subject/description.
func SetDescription(ctx context.Context, client *whatsmeow.Client, groupJID types.JID, description string) error {
	return client.SetGroupTopic(ctx, groupJID, "", "", description)
}

// SetName updates a group's name.
func SetName(ctx context.Context, client *whatsmeow.Client, groupJID types.JID, name string) error {
	return client.SetGroupName(ctx, groupJID, name)
}

// Leave leaves a group.
func Leave(ctx context.Context, client *whatsmeow.Client, groupJID types.JID) error {
	return client.LeaveGroup(ctx, groupJID)
}

// ListJoined returns all joined groups for the session.
func ListJoined(ctx context.Context, client *whatsmeow.Client) ([]*types.GroupInfo, error) {
	return client.GetJoinedGroups(ctx)
}
