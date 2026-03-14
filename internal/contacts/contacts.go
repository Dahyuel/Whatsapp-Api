package contacts

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// CheckExists verifies whether one or more phone numbers are registered on WhatsApp.
func CheckExists(ctx context.Context, client *whatsmeow.Client, phones []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, phone := range phones {
		jids, err := client.IsOnWhatsApp([]string{phone})
		if err != nil {
			return nil, fmt.Errorf("check number %s: %w", phone, err)
		}
		for _, j := range jids {
			result[phone] = j.IsIn
		}
	}
	return result, nil
}

// GetContactInfo returns contact info for a JID (avatar URL, about/status).
func GetContactInfo(ctx context.Context, client *whatsmeow.Client, jid types.JID) (map[string]interface{}, error) {
	info := map[string]interface{}{
		"jid": jid.String(),
	}

	// Profile picture
	pic, err := client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err == nil && pic != nil {
		info["profile_picture_url"] = pic.URL
	}

	// User info (status)
	userInfo, err := client.GetUserInfo([]types.JID{jid})
	if err == nil {
		for _, u := range userInfo {
			info["status"] = u.Status
			info["status_at"] = u.StatusAt
			info["verified"] = u.VerifiedName
		}
	}

	return info, nil
}

// GetContacts returns all stored contacts.
func GetContacts(ctx context.Context, client *whatsmeow.Client) (interface{}, error) {
	contacts, err := client.Store.Contacts.GetAllContacts()
	if err != nil {
		return nil, fmt.Errorf("get contacts: %w", err)
	}
	return contacts, nil
}

// BlockContact blocks a contact.
func BlockContact(ctx context.Context, client *whatsmeow.Client, jid types.JID) error {
	return client.UpdateBlocklist(jid, whatsmeow.BlocklistChangeActionBlock)
}

// UnblockContact unblocks a contact.
func UnblockContact(ctx context.Context, client *whatsmeow.Client, jid types.JID) error {
	return client.UpdateBlocklist(jid, whatsmeow.BlocklistChangeActionUnblock)
}

// GetProfilePhoto returns the profile picture URL for a JID.
func GetProfilePhoto(ctx context.Context, client *whatsmeow.Client, jid types.JID) (string, error) {
	pic, err := client.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{Preview: false})
	if err != nil {
		return "", fmt.Errorf("get profile picture: %w", err)
	}
	if pic == nil {
		return "", nil
	}
	return pic.URL, nil
}
