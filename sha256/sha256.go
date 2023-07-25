package sha256

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

type Cipher struct {
	source string
	salt   uint64
}

// SetString 원본 소스 문자열
func (cp *Cipher) SetString(str string) {
	cp.source = str
}

// SetSalt 소금칠을 할 경우 소금값
func (cp *Cipher) SetSalt(s uint64) {
	cp.salt = s
}

// Encode sha 256 으로 인코딩
func (cp *Cipher) Encode() string {
	bytes := []byte(cp.source)
	if cp.salt > 0 {
		saltByte := make([]byte, 8)
		binary.LittleEndian.PutUint64(saltByte, cp.salt)

		bytes = append(bytes, saltByte...)
	}
	hash := sha256.New()
	hash.Write(bytes)

	s := hash.Sum(nil)

	return hex.EncodeToString(s)
}
