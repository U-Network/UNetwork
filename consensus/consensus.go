package consensus

import (
	"UNetwork/common/log"
	"fmt"
	"time"
)

type ConsensusService interface {
	Start() error
	Halt() error
}

func Log(message string) {
	logMsg := fmt.Sprintf("[%s] %s", time.Now().Format("02/01/2006 15:04:05"), message)
	fmt.Println(logMsg)
	log.Info(logMsg)
}
