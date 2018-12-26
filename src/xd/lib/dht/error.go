package dht

import (
	"fmt"
	"github.com/zeebo/bencode"
)

type Error struct {
	Code    int64
	Message string
}

func (e *Error) MarshalBencode() ([]byte, error) {
	return bencode.EncodeBytes([]interface{}{
		e.Code,
		e.Message,
	})
}

func (e *Error) UnmarshalBencode(d []byte) error {
	var dec []interface{}
	err := bencode.DecodeBytes(d, &dec)
	if err != nil {
		return err
	}
	if len(dec) != 2 {
		return fmt.Errorf("bad size of error: %d", len(dec))
	}
	var ok bool
	e.Code, ok = dec[0].(int64)
	if !ok {
		return fmt.Errorf("first element is not an int")
	}
	e.Message, ok = dec[1].(string)
	if !ok {
		return fmt.Errorf("second element is not a string")
	}
	return nil
}

const ErrCodeGeneric = 201
const ErrCodeServer = 202
const ErrCodeProtocol = 203
const ErrCodeMethod = 204
