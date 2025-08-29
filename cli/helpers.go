package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"golang.org/x/term"
)

func GetVaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(home, ".go-vault")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(dir, "vault.data"), nil
}

func ReadPassword(prompt string) ([]byte, error) {
	fmt.Println(prompt)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()

	return pw, err
}

func ReadPasswordMasked(prompt string) []byte {
	fmt.Print(prompt)
	fd := int(os.Stdin.Fd())
	state, _ := term.MakeRaw(fd)
	defer term.Restore(fd, state)

	var input []rune
	for {
		var buf [1]byte
		os.Stdin.Read(buf[:])
		c := buf[0]

		switch c {
		case 13, 10: // Enter
			fmt.Println()
			return []byte(string(input))
		case 127, 8: // Backspace
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b")
			}
		default:
			r, _ := utf8.DecodeRune(buf[:])
			input = append(input, r)
			fmt.Print("*")
		}
	}
}
