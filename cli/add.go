package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fahmaliyi/vault/vault"
	"github.com/google/uuid"
	"golang.org/x/term"
)

func AddEntryCLI(v *vault.Vault) {
	// Temporarily suspend Bubble Tea
	fmt.Print("\n--- Add New Entry ---\n")

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Title: ")
	title, _ := reader.ReadString('\n')
	title = strings.TrimSpace(title)

	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Secret: ")
	secretBytes, _ := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	secret := strings.TrimSpace(string(secretBytes))

	fmt.Print("Notes (optional): ")
	notes, _ := reader.ReadString('\n')
	notes = strings.TrimSpace(notes)

	e := vault.Entry{
		ID:       uuid.New().String(),
		Title:    title,
		Username: username,
		Secret:   []byte(secret),
		Notes:    notes,
	}

	v.Add(e)
	v.Save()

	fmt.Println("Entry added successfully!\nPress Enter to continue...")
	reader.ReadString('\n') // wait for user
	vault.Zero([]byte(secret))
}
