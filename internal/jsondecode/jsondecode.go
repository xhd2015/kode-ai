package jsondecode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func UnmarshalSafe(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := dec.Decode(&v)
	if err != nil {
		return fmt.Errorf("cannot compress body, invalid json: %v", err)
	}

	var noMore interface{}
	noMoreErr := dec.Decode(&noMore)
	if noMoreErr != nil {
		if noMoreErr != io.EOF {
			return fmt.Errorf("invalid json: %v", noMoreErr)
		}
	} else {
		err := json.Unmarshal([]byte(data), &noMore)
		if err != nil {
			return fmt.Errorf("invalid json: %v", err)
		} else {
			return fmt.Errorf("invalid json: multiple json object found")
		}
	}
	return nil
}

func UnmarshalSafeAny(data []byte) (interface{}, error) {
	var v interface{}

	err := UnmarshalSafe(data, &v)
	if err != nil {
		return nil, err
	}
	return v, nil
}
