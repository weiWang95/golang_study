package base64

import (
	"bytes"
)

const _charSet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_"

func Base64(s string) string {
	arr := []byte(s)
	var buf bytes.Buffer

	for _, v := range arr {
		buf.WriteByte(_charSet[v%64])
	}

	return buf.String()
}
