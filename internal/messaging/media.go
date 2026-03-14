package messaging

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

// MediaType represents the kind of media being sent.
type MediaType string

const (
	MediaImage    MediaType = "image"
	MediaVideo    MediaType = "video"
	MediaAudio    MediaType = "audio"
	MediaDocument MediaType = "document"
	MediaSticker  MediaType = "sticker"
)

// UploadAndBuildMedia uploads raw bytes to WhatsApp and returns the message proto.
func UploadAndBuildMedia(ctx context.Context, client *whatsmeow.Client, data []byte, mtype MediaType, mimeType, caption, filename string) (*waProto.Message, error) {
	var wmType whatsmeow.MediaType
	switch mtype {
	case MediaImage:
		wmType = whatsmeow.MediaImage
	case MediaVideo:
		wmType = whatsmeow.MediaVideo
	case MediaAudio:
		wmType = whatsmeow.MediaAudio
	case MediaDocument:
		wmType = whatsmeow.MediaDocument
	case MediaSticker:
		wmType = whatsmeow.MediaImage
	default:
		wmType = whatsmeow.MediaDocument
	}

	uploaded, err := client.Upload(ctx, data, wmType)
	if err != nil {
		return nil, fmt.Errorf("upload media: %w", err)
	}

	switch mtype {
	case MediaImage:
		return &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Url:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				Caption:       proto.String(caption),
			},
		}, nil

	case MediaVideo:
		return &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Url:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				Caption:       proto.String(caption),
			},
		}, nil

	case MediaAudio:
		return &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Url:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				Ptt:           proto.Bool(mimeType == "audio/ogg; codecs=opus"),
			},
		}, nil

	case MediaSticker:
		return &waProto.Message{
			StickerMessage: &waProto.StickerMessage{
				Url:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
			},
		}, nil

	default: // Document
		return &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Url:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimeType),
				FileEncSha256: uploaded.FileEncSHA256,
				FileSha256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uint64(len(data))),
				FileName:      proto.String(filename),
				Caption:       proto.String(caption),
			},
		}, nil
	}
}

// FetchURL downloads the content of a URL and returns the bytes.
func FetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// SaveMedia saves downloaded media bytes to a storage path.
func SaveMedia(storagePath, sessionID, msgID, ext string, data []byte) (string, error) {
	dir := filepath.Join(storagePath, sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s_%d%s", msgID, time.Now().UnixMilli(), ext)
	path := filepath.Join(dir, filename)
	return path, os.WriteFile(path, data, 0644)
}
