package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/fahmaliyi/vault/vault"
	"github.com/google/uuid"
)

func RunCommands(v *vault.Vault) {
	reader := bufio.NewReader(os.Stdin)
	var idMap map[int]string

	for {
		fmt.Println("\nCommands: a=add, l=list, s N=show, c N=copy, d N=delete, q=quit")
		fmt.Print("> ")

		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		cmd := parts[0]

		switch cmd {
		case "a":
			handleAdd(v, reader)
			idMap = nil
		case "l":
			idMap = handleList(v)
		case "s", "c", "d":
			if len(parts) < 2 {
				fmt.Println("Specify item number")
				continue
			}
			var num int
			fmt.Sscanf(parts[1], "%d", &num)
			id, ok := idMap[num]
			if !ok {
				fmt.Println("Invalid item number")
				continue
			}
			switch cmd {
			case "s":
				handleShow(v, id)
			case "c":
				handleCopy(v, id)
			case "d":
				handleDelete(v, id)
			}
		case "q":
			fmt.Println("Exiting.")
			return
		default:
			fmt.Println("Unknown command")
		}
	}
}

// --- Individual command handlers ---

func handleAdd(v *vault.Vault, reader *bufio.Reader) {
	e := vault.Entry{ID: uuid.New().String()}

	fmt.Print("Title: ")
	e.Title, _ = reader.ReadString('\n')
	e.Title = strings.TrimSpace(e.Title)

	fmt.Print("Username: ")
	e.Username, _ = reader.ReadString('\n')
	e.Username = strings.TrimSpace(e.Username)

	secret := ReadPasswordMasked("Secret: ")
	e.Secret = secret

	fmt.Print("Notes: ")
	e.Notes, _ = reader.ReadString('\n')
	e.Notes = strings.TrimSpace(e.Notes)

	v.Add(e)
	if err := v.Save(); err != nil {
		fmt.Println("Error saving vault:", err)
	} else {
		fmt.Println("Entry added!")
	}
}

func handleList(v *vault.Vault) map[int]string {
	entries := v.List()
	fmt.Println("Vault entries:")
	idMap := make(map[int]string)
	for i, e := range entries {
		num := i + 1
		idMap[num] = e.ID
		fmt.Printf("%d) Title: %s | Username: %s\n", num, e.Title, e.Username)
	}
	return idMap
}

func handleShow(v *vault.Vault, id string) {
	e := v.Get(id)
	if e == nil {
		fmt.Println("Entry not found")
		return
	}
	fmt.Printf("Title: %s\nUsername: %s\nSecret: %s\nNotes: %s\n",
		e.Title, e.Username, string(e.Secret), e.Notes)
}

func handleCopy(v *vault.Vault, id string) {
	e := v.Get(id)
	if e == nil {
		fmt.Println("Entry not found")
		return
	}
	clipboard.WriteAll(string(e.Secret))
	fmt.Println("Secret copied to clipboard. Clearing in 30 seconds...")
	time.AfterFunc(30*time.Second, func() {
		clipboard.WriteAll("")
	})
}

func handleDelete(v *vault.Vault, id string) {
	v.Delete(id)
	if err := v.Save(); err != nil {
		fmt.Println("Error saving vault:", err)
	} else {
		fmt.Println("Entry deleted!")
	}
}
