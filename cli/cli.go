package cli

import (
	"math/rand"
	"time"

	"UGCNetwork/common/config"
	"UGCNetwork/common/log"
	"UGCNetwork/crypto"
)

func init() {
	log.Init()
	crypto.SetAlg(config.Parameters.EncryptAlg)
	//seed transaction nonce
	rand.Seed(time.Now().UnixNano())
}
