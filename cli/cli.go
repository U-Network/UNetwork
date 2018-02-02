package cli

import (
	"math/rand"
	"time"

	"UNetwork/common/config"
	"UNetwork/common/log"
	"UNetwork/crypto"
)

func init() {
	log.Init()
	crypto.SetAlg(config.Parameters.EncryptAlg)
	//seed transaction nonce
	rand.Seed(time.Now().UnixNano())
}
