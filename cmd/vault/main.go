package main

import (
	"fmt"
	"os"

	"github.com/fahmaliyi/vault/cli"
	"github.com/fahmaliyi/vault/vault"
)

func main() {
	vaultPath, err := cli.GetVaultPath()
	if err != nil {
		fmt.Println("Error determining vault path:", err)
		return
	}
	v := vault.NewVault(vaultPath, nil)

	syncer := &vault.GoogleDriveSync{}
	v.SetSyncer(syncer)

	var master []byte
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		fmt.Println("No vault found. Setting up new master password.")
		master = cli.ReadPasswordMasked("Set master password: ")

		if err := v.Create(master); err != nil {
			fmt.Println("Error creating vault:", err)
			return
		}
	} else {
		master = cli.ReadPasswordMasked("Enter master password: ")

		if err := v.Open(master); err != nil {
			fmt.Println("Error opening vault:", err)
			return
		}
	}
	defer vault.Zero(master)

	// Start the CLI loop
	cli.RunCommands(v)
}
