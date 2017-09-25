package signalset

import (
	"fmt"
	"os"
)

type SignalHandler func(s os.Signal, v interface{})

type SignalSet struct {
	signalMap map[os.Signal]SignalHandler
}

func New() *SignalSet {
	return &SignalSet{
		signalMap: make(map[os.Signal]SignalHandler),
	}
}

func (s *SignalSet) Register(signal os.Signal, handle SignalHandler) {
	if _, ok := s.signalMap[signal]; !ok {
		s.signalMap[signal] = handle
	}
}

func (s *SignalSet) Handle(signal os.Signal, v interface{}) error {
	if handler, ok := s.signalMap[signal]; ok {
		handler(signal, v)
		return nil
	} else {
		return fmt.Errorf("No handler available for signalset %v", signal)
	}
}
