package ethereum

import (
	"github.com/U-Network/UNetwork/global"
	"path/filepath"
)

func ReadBlockConfigDir() string {
	sdir := global.Homedir()
	sdir = filepath.Join(sdir, "config")
	extraFile := filepath.Join(sdir, "eth_block_extra.line")
	lineData, err := global.ReadFile(extraFile)
	if err != nil {
		return ""
	}
	return lineData
}
