package core

import (
	"os"
	"path/filepath"
)

const StateDir = ".unetwork/state"

func Homedir() string {
	return os.ExpandEnv(filepath.Join("$HOME", StateDir))
}
