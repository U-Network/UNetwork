package message

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"

	"UNetwork/common/serialization"
	"UNetwork/core/ledger"
	"UNetwork/events"
	. "UNetwork/net/protocol"
	."UNetwork/common"
	"UNetwork/common/log"
)

type ChatPayload struct {
	Address  string
	UserName string
	Content  []byte
	Nonce    uint64
}

type chat struct {
	msgHdr
	ChatPayload
}

func (msg *ChatPayload)Hash() Uint256{
	var msgHash Uint256
	buffer := bytes.NewBuffer([]byte{})
	if err := msg.Serialization(buffer);err != nil {
		log.Error("chat message serialization error")
		return msgHash
	}
	temp := sha256.Sum256(buffer.Bytes())
	msgHash = Uint256(sha256.Sum256(temp[:]))

	return msgHash
}

func NewChatMsg(p *ChatPayload) ([]byte, error) {
	var c chat
	c.ChatPayload = *p
	buf := new(bytes.Buffer)
	p.Serialization(buf)
	b := new(bytes.Buffer)
	if err := binary.Write(b, binary.LittleEndian, buf.Bytes()); err != nil {
		return nil, err
	}
	s := checkSum(b.Bytes())
	c.init("chat", s, uint32(len(b.Bytes())))
	m, err := c.Serialization()
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (msg chat) Verify(buf []byte) error {
	return msg.msgHdr.Verify(buf)
}

func (msg chat) Handle(node Noder) error {
	payload := &msg.ChatPayload
	if !node.LocalNode().ExistedID(payload.Hash()) {
		node.LocalNode().Relay(node, payload)
		ledger.DefaultLedger.Blockchain.BCEvents.Notify(events.EventChatMessage, &msg.ChatPayload)
	}

	return nil
}

func (payload ChatPayload) Serialization(w io.Writer) error {
	if err := serialization.WriteVarString(w, payload.Address); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, payload.UserName); err != nil {
		return err
	}
	if err := serialization.WriteVarBytes(w, payload.Content); err != nil {
		return err
	}
	if err := serialization.WriteUint64(w, payload.Nonce); err != nil {
		return err
	}

	return nil
}

func (payload *ChatPayload) Deserialization(r io.Reader) error {
	var err error
	payload.Address, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}
	payload.UserName, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}
	payload.Content, err = serialization.ReadVarBytes(r)
	if err != nil {
		return err
	}
	payload.Nonce, err = serialization.ReadUint64(r)
	if err != nil {
		return err
	}

	return nil
}

func (msg chat) Serialization() ([]byte, error) {
	hdrBuf, err := msg.msgHdr.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	msg.ChatPayload.Serialization(buf)

	return buf.Bytes(), nil
}

func (msg *chat) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	if err := binary.Read(buf, binary.LittleEndian, &(msg.msgHdr)); err != nil {
		return err
	}
	msg.ChatPayload.Deserialization(buf)

	return nil
}
