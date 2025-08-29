package vault

type Syncer interface {
	// Pull downloads the latest encrypted vault from remote
	Pull(vaultPath string) error

	// Push uploads the local encrypted vault to remote
	Push(vaultPath string) error
}
