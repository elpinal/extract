// +build gofuzz

package extract

import "bytes"

func Fuzz(data []byte) int {
	b := bytes.NewBuffer(data)
	if _, _, err := Extract(b); err != nil {
		return 0
	}
	return 1
}
