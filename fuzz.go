package extract

import "bytes"

// +build gofuzz

func Fuzz(data []byte) int {
	b := bytes.NewBuffer(data)
	if _, _, err := Extract(b); err != nil {
		return 0
	}
	return 1
}
