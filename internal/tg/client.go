package tg

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
)

// Client wraps the gotd Telegram client to provide high-level operations
// for the Saved Messages Manager.
type Client struct {
	client *telegram.Client
	api    *tg.Client
	User   *tg.User
}

// NewClient creates a new Telegram client.
// It requires APP_ID and APP_HASH from environment variables.
func NewClient() (*Client, error) {
	appID := os.Getenv("TG_APP_ID")
	appHash := os.Getenv("TG_APP_HASH")

	if appID == "" || appHash == "" {
		return nil, errors.New("TG_APP_ID or TG_APP_HASH environment variables not set")
	}

	// We use a custom session storage or default file storage "session.json"
	// For simplicity, we'll let gotd handle the storage in the current directory.

	// Note: We are not initializing the client connection here fully,
	// just setting up the struct. The actual connection happens in Run.
	// However, to make it usable as a service, we often want to wrap the client lifecycle.

	// For this simple app, we will expose a Run method that starts the client
	// and keeps it running, or we can use the library's recommended pattern.

	return &Client{}, nil // Real initialization happens in Start
}

// StartAndListen connects to Telegram and blocks.
// It executes the 'onReady' callback when the client is authenticated and ready to query.
func (c *Client) StartAndListen(ctx context.Context, onReady func(ctx context.Context) error) error {
	appID := os.Getenv("TG_APP_ID")
	appHash := os.Getenv("TG_APP_HASH")
	if appID == "" || appHash == "" {
		return errors.New("missing TG_APP_ID/TG_APP_HASH")
	}

	// Basic session file
	sessionDir := "session"
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return err
	}

	appIDInt, err := strconv.Atoi(appID)
	if err != nil {
		return fmt.Errorf("invalid TG_APP_ID: %w", err)
	}

	client := telegram.NewClient(appIDInt, appHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: "session/session.json",
		},
	})

	c.client = client

	for {
		err := client.Run(ctx, func(ctx context.Context) error {
			// Auth flow
			status, err := client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("auth status error: %w", err)
			}

			if !status.Authorized {
				flow := auth.NewFlow(termAuth{}, auth.SendCodeOptions{})
				if err := client.Auth().IfNecessary(ctx, flow); err != nil {
					return fmt.Errorf("auth error: %w", err)
				}
			}

			// Get self
			self, err := client.Self(ctx)
			if err != nil {
				return fmt.Errorf("failed to get self: %w", err)
			}
			c.User = self
			c.api = client.API()

			fmt.Printf("Logged in as %s %s (@%s)\n", self.FirstName, self.LastName, self.Username)

			return onReady(ctx)
		})

		if err != nil {
			if strings.Contains(err.Error(), "AUTH_RESTART") { // Check for AUTH_RESTART error
				fmt.Println("Received AUTH_RESTART. Deleting session and restarting...")
				// Delete session file to force re-auth
				if rErr := os.RemoveAll("session"); rErr != nil {
					fmt.Printf("failed to remove session directory: %v\n", rErr)
				}
				// Recreate client is a bit tricky here because we created it outside loop.
				// But client.Run should be restartable if we just loop?
				// Actually, `client` struct might hold some state.
				// Better to create a new client instance?
				// The current structure creates `client` variable outside.
				// Let's just recreate the client inside the loop if we want to be safe,
				// but StartAndListen takes the struct receiver `c`.
				// The `c.client` is assigned.

				// Ideally we should rebuild the client.
				// Let's modify the code to rebuild client inside the loop if needed.
				// But we are inside StartAndListen.

				// Simplest valid "Retry" is to:
				// 1. Delete session.
				// 2. Return error? No, we want to stay alive.

				// If we just continue the loop, we call `client.Run` again on the *same* client instance.
				// GOTD client might be in a closed state.
				// Let's try to re-initialize the client variable.

				newClient := telegram.NewClient(appIDInt, appHash, telegram.Options{
					SessionStorage: &telegram.FileSessionStorage{
						Path: "session/session.json",
					},
				})
				c.client = newClient
				client = newClient

				continue
			}
			return err
		}

		// If no error, it means ctx was canceled or normal exit
		return nil
	}
}

type termAuth struct{}

func (termAuth) Phone(_ context.Context) (string, error) {
	fmt.Print("Enter phone: ")
	phone, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(phone), nil
}

func (termAuth) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (termAuth) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("signup not supported")
}

func (termAuth) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func (termAuth) Password(ctx context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	pass, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pass), nil
}

// MediaItem represents a single media attachment
type MediaItem struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

type WebPagePreview struct {
	SiteName    string `json:"site_name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// SavedMessage represents a simplified view of a Telegram message
type SavedMessage struct {
	ID          int             `json:"id"`  // The main ID (usually the first encountered or one with text)
	IDs         []int           `json:"ids"` // All IDs in this group (for deletion)
	Date        int             `json:"date"`
	Message     string          `json:"message"`
	MediaType   string          `json:"media_type,omitempty"` // For backward compatibility / single media
	Attachments []MediaItem     `json:"attachments,omitempty"`
	GroupedID   int64           `json:"grouped_id,omitempty"`
	WebPreview  *WebPagePreview `json:"web_preview,omitempty"`
}

// GetSavedMessages fetches the history of 'Saved Messages' (InputPeerSelf).
func (c *Client) GetSavedMessages(ctx context.Context, offsetID int, limit int, addOffset int) ([]SavedMessage, int, error) {
	if c.api == nil {
		return nil, 0, errors.New("client not initialized")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	history, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:      &tg.InputPeerSelf{},
		OffsetID:  offsetID,
		Limit:     limit,
		AddOffset: addOffset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get history: %w", err)
	}

	var messages []tg.MessageClass
	var totalCount int

	switch h := history.(type) {
	case *tg.MessagesMessages:
		messages = h.Messages
		totalCount = len(messages)
		fmt.Printf("[DEBUG] Got MessagesMessages. Count: %d\n", totalCount)
	case *tg.MessagesMessagesSlice:
		messages = h.Messages
		totalCount = h.Count
		fmt.Printf("[DEBUG] Got MessagesMessagesSlice. Count: %d, Len: %d\n", totalCount, len(messages))
	case *tg.MessagesChannelMessages:
		messages = h.Messages
		totalCount = h.Count
		fmt.Printf("[DEBUG] Got MessagesChannelMessages. Count: %d\n", totalCount)
	default:
		return nil, 0, fmt.Errorf("unexpected history type: %T", history)
	}

	var result []SavedMessage

	// Messages usually come new to old.
	// Grouped messages (albums) are adjacent.

	for _, msg := range messages {
		m, ok := msg.(*tg.Message)
		if !ok {
			continue
		}

		mediaType := ""
		var webPreview *WebPagePreview

		if m.Media != nil {
			switch media := m.Media.(type) {
			case *tg.MessageMediaPhoto:
				mediaType = "Photo"
			case *tg.MessageMediaDocument:
				mediaType = "Document"
			case *tg.MessageMediaWebPage:
				mediaType = "WebLink"
				if wp, ok := media.Webpage.(*tg.WebPage); ok {
					webPreview = &WebPagePreview{
						SiteName:    wp.SiteName,
						Title:       wp.Title,
						Description: wp.Description,
						URL:         wp.URL,
					}
				}
			default:
				mediaType = "Media"
			}
		}

		// Logic to merge with previous if GroupedID matches
		// Note: 'previous' in 'result' is actually a NEWER message because of iteration order.
		// If we encounter a message that belongs to the same group as the last added message,
		// we merge it into that one.

		merged := false
		if m.GroupedID != 0 && len(result) > 0 {
			lastIdx := len(result) - 1
			last := &result[lastIdx]

			if last.GroupedID == m.GroupedID {
				// Merge into last
				merged = true
				last.IDs = append(last.IDs, m.ID)

				// Keep text if current has it and last didn't (or append? usually caption is on one)
				if last.Message == "" && m.Message != "" {
					last.Message = m.Message
				}

				// Copy media to attachments
				if mediaType == "Photo" || mediaType == "Document" { // Only attach renderable types
					last.Attachments = append(last.Attachments, MediaItem{
						ID:   m.ID,
						Type: mediaType,
					})
				}
			}
		}

		if !merged {
			// Create new
			item := SavedMessage{
				ID:          m.ID,
				IDs:         []int{m.ID},
				Date:        m.Date,
				Message:     m.Message,
				MediaType:   mediaType, // Keep for single display or fallback
				GroupedID:   m.GroupedID,
				Attachments: []MediaItem{},
				WebPreview:  webPreview,
			}

			// If it has media, add to attachments too for consistency
			if mediaType == "Photo" || mediaType == "Document" {
				item.Attachments = append(item.Attachments, MediaItem{
					ID:   m.ID,
					Type: mediaType,
				})
			}

			result = append(result, item)
		}
	}

	return result, totalCount, nil
}

// DeleteMessages deletes messages by ID from Saved Messages.
func (c *Client) DeleteMessages(ctx context.Context, ids []int) error {
	if c.api == nil {
		return errors.New("client not initialized")
	}

	if len(ids) == 0 {
		return nil
	}

	_, err := c.api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
		Revoke: true,
		ID:     ids,
	})

	return err
}

// GetMessageMedia downloads the media for a given message ID.
func (c *Client) GetMessageMedia(ctx context.Context, msgID int) ([]byte, string, error) {
	if c.api == nil {
		return nil, "", errors.New("client not initialized")
	}

	// 1. Get the message
	msgs, err := c.api.MessagesGetMessages(ctx, []tg.InputMessageClass{
		&tg.InputMessageID{ID: msgID},
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get message: %w", err)
	}

	var msg *tg.Message
	switch m := msgs.(type) {
	case *tg.MessagesMessages:
		if len(m.Messages) > 0 {
			if mm, ok := m.Messages[0].(*tg.Message); ok {
				msg = mm
			}
		}
	case *tg.MessagesMessagesSlice:
		if len(m.Messages) > 0 {
			if mm, ok := m.Messages[0].(*tg.Message); ok {
				msg = mm
			}
		}
	case *tg.MessagesChannelMessages:
		if len(m.Messages) > 0 {
			if mm, ok := m.Messages[0].(*tg.Message); ok {
				msg = mm
			}
		}
	}

	if msg == nil || msg.Media == nil {
		return nil, "", errors.New("message media not found")
	}

	// 2. Determine location and content type
	var location tg.InputFileLocationClass
	contentType := "application/octet-stream"

	switch media := msg.Media.(type) {
	case *tg.MessageMediaPhoto:
		contentType = "image/jpeg"
		photo, ok := media.Photo.(*tg.Photo)
		if !ok {
			return nil, "", errors.New("photo is empty or not *tg.Photo")
		}

		var bestSize string
		// Priority: w (large), y (large), x (medium), m (small), s (small)
		// Or progressive sizes (i, j?)
		// Let's iterate and see what we have.
		// We prefer 'y' or 'w' or 'x'.
		for _, s := range photo.Sizes {
			if sz, ok := s.(*tg.PhotoSize); ok {
				// Log what we see
				fmt.Printf("[DEBUG] Photo %d size: %s (%dx%d)\n", photo.ID, sz.Type, sz.W, sz.H)
				if sz.Type == "w" || sz.Type == "y" {
					bestSize = sz.Type
					break
				}
				if sz.Type == "x" {
					bestSize = sz.Type // Keep looking for w/y but x is good
				}
			}
			if sz, ok := s.(*tg.PhotoSizeProgressive); ok {
				fmt.Printf("[DEBUG] Photo %d progressive size: %s (%dx%d)\n", photo.ID, sz.Type, sz.W, sz.H)
				if sz.Type == "w" || sz.Type == "y" {
					bestSize = sz.Type
					break
				}
				if sz.Type == "x" {
					bestSize = sz.Type
				}
			}
		}

		// Fallback to last one if nothing standard found (e.g. only thumbs)
		if bestSize == "" && len(photo.Sizes) > 0 {
			last := photo.Sizes[len(photo.Sizes)-1]
			if sz, ok := last.(*tg.PhotoSize); ok {
				bestSize = sz.Type
			}
			if sz, ok := last.(*tg.PhotoSizeProgressive); ok {
				bestSize = sz.Type
			}
		}

		if bestSize == "" {
			return nil, "", fmt.Errorf("no suitable photo size found for photo %d", photo.ID)
		}

		fmt.Printf("[DEBUG] Selected size '%s' for photo %d\n", bestSize, photo.ID)

		location = &tg.InputPhotoFileLocation{
			ID:            photo.ID,
			AccessHash:    photo.AccessHash,
			FileReference: photo.FileReference,
			ThumbSize:     bestSize,
		}

	case *tg.MessageMediaDocument:
		doc, ok := media.Document.(*tg.Document)
		if !ok {
			return nil, "", errors.New("document is not *tg.Document")
		}
		contentType = doc.MimeType
		location = &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
			ThumbSize:     "",
		}

	case *tg.MessageMediaWebPage:
		wp, ok := media.Webpage.(*tg.WebPage)
		if !ok {
			return nil, "", errors.New("webpage is empty or pending")
		}
		if wp.Photo == nil {
			return nil, "", errors.New("webpage has no photo")
		}

		contentType = "image/jpeg"
		photo, ok := wp.Photo.(*tg.Photo)
		if !ok {
			return nil, "", errors.New("webpage photo is not *tg.Photo")
		}

		// Reusing photo logic
		var bestSize string
		for _, s := range photo.Sizes {
			if sz, ok := s.(*tg.PhotoSize); ok {
				if sz.Type == "w" || sz.Type == "y" {
					bestSize = sz.Type
					break
				}
				if sz.Type == "x" {
					bestSize = sz.Type
				}
			}
			if sz, ok := s.(*tg.PhotoSizeProgressive); ok {
				if sz.Type == "w" || sz.Type == "y" {
					bestSize = sz.Type
					break
				}
				if sz.Type == "x" {
					bestSize = sz.Type
				}
			}
		}

		if bestSize == "" && len(photo.Sizes) > 0 {
			last := photo.Sizes[len(photo.Sizes)-1]
			if sz, ok := last.(*tg.PhotoSize); ok {
				bestSize = sz.Type
			}
			if sz, ok := last.(*tg.PhotoSizeProgressive); ok {
				bestSize = sz.Type
			}
		}

		if bestSize == "" {
			return nil, "", fmt.Errorf("no suitable photo size found for webpage photo %d", photo.ID)
		}

		location = &tg.InputPhotoFileLocation{
			ID:            photo.ID,
			AccessHash:    photo.AccessHash,
			FileReference: photo.FileReference,
			ThumbSize:     bestSize,
		}

	default:
		return nil, "", fmt.Errorf("unsupported media type: %T", msg.Media)
	}

	// 3. Download
	d := downloader.NewDownloader()
	data := bytes.NewBuffer(nil)

	_, err = d.Download(c.api, location).Stream(ctx, data)
	if err != nil {
		return nil, "", fmt.Errorf("download failed: %w", err)
	}

	if data.Len() == 0 {
		return nil, "", fmt.Errorf("downloaded 0 bytes for message %d", msgID)
	}

	return data.Bytes(), contentType, nil
}
