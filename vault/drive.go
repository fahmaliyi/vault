package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

const VaultFileName = "vault.dat"

type GoogleDriveSync struct {
	token *oauth2.Token
}

// loadToken reads the OAuth2 token from ~/.go-vault/token.json
func loadToken() (*oauth2.Token, error) {
	path := fmt.Sprintf("%s/.go-vault/token.json", os.Getenv("HOME"))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// sync performs pull (download) or push (upload) of the encrypted vault
func (g *GoogleDriveSync) sync(vaultPath string, upload bool) error {
	ctx := context.Background()

	credPath := fmt.Sprintf("%s/.go-vault/credentials.json", os.Getenv("HOME"))
	b, err := os.ReadFile(credPath)
	if err != nil {
		return fmt.Errorf("failed to read credentials: %v", err)
	}

	// Parse Google OAuth2 config from JSON
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("failed to parse credentials: %v", err)
	}

	// Create client with token
	client := config.Client(ctx, g.token)

	// Initialize Drive service
	srv, err := drive.New(client)
	if err != nil {
		return fmt.Errorf("failed to create Drive service: %v", err)
	}

	// Look for existing vault file
	r, err := srv.Files.List().Q(fmt.Sprintf("name='%s'", VaultFileName)).Do()
	if err != nil {
		return fmt.Errorf("failed to query Drive: %v", err)
	}

	var fileID string
	if len(r.Files) > 0 {
		fileID = r.Files[0].Id
	}

	if upload {
		data, err := os.ReadFile(vaultPath)
		if err != nil {
			return fmt.Errorf("failed to read local vault: %v", err)
		}

		if fileID == "" {
			// Create new file
			f := &drive.File{Name: VaultFileName}
			_, err := srv.Files.Create(f).Media(bytes.NewReader(data)).Do()
			if err != nil {
				return fmt.Errorf("failed to upload vault: %v", err)
			}
		} else {
			// Update existing file
			_, err := srv.Files.Update(fileID, nil).Media(bytes.NewReader(data)).Do()
			if err != nil {
				return fmt.Errorf("failed to update vault: %v", err)
			}
		}
	} else {
		if fileID == "" {
			return fmt.Errorf("no remote vault found on Google Drive")
		}
		resp, err := srv.Files.Get(fileID).Download()
		if err != nil {
			return fmt.Errorf("failed to download vault: %v", err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read downloaded vault: %v", err)
		}

		err = os.WriteFile(vaultPath, data, 0600)
		if err != nil {
			return fmt.Errorf("failed to write local vault: %v", err)
		}
	}

	return nil
}

// Pull downloads the remote vault from Google Drive
func (g *GoogleDriveSync) Pull(vaultPath string) error {
	if g.token == nil {
		tok, _ := loadToken()
		g.token = tok
	}
	return g.sync(vaultPath, false)
}

// Push uploads the local vault to Google Drive
func (g *GoogleDriveSync) Push(vaultPath string) error {
	if g.token == nil {
		tok, _ := loadToken()
		g.token = tok
	}
	return g.sync(vaultPath, true)
}
