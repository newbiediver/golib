package aes256

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"log"
)

// Cipher 는 AES 256bit 를 위한 핸들러
type Cipher struct {
	key string
	iv  string
}

// SetKey : 256bit(32bytes) 크기의 키를 셋팅합니다.
func (cp *Cipher) SetKey(key string) error {
	if len(key) != 32 {
		return fmt.Errorf("Key must be set 256 bit")
	}

	cp.key = key

	return nil
}

// SetIV : 128bit(16byte) 크기의 Init Vector 값을 셋팅합니다.
func (cp *Cipher) SetIV(iv string) error {
	if len(iv) != 16 {
		return fmt.Errorf("IV must be set 128 bit")
	}

	cp.iv = iv

	return nil
}

// Encode : 암호화!
func (cp *Cipher) Encode(data []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(cp.key))
	if err != nil {
		return nil, err
	}

	cipherBlock := encrypt(block, data, cp.iv)

	return cipherBlock, nil
}

func encrypt(b cipher.Block, plain []byte, iv string) []byte {
	mod := len(plain) % aes.BlockSize
	paddingSize := aes.BlockSize - mod
	padding := make([]byte, paddingSize)
	for i := 0; i < paddingSize; i++ {
		padding[i] = byte(paddingSize)
	}
	plain = append(plain, padding...)

	cipherBytes := make([]byte, len(plain))
	mode := cipher.NewCBCEncrypter(b, []byte(iv))
	mode.CryptBlocks(cipherBytes, plain)

	return cipherBytes
}

// Decode : 복호화!
func (cp *Cipher) Decode(data []byte) ([]byte, error) {
	block, err := aes.NewCipher([]byte(cp.key))
	if err != nil {
		return nil, err
	}

	cipherBlock, err := decrypt(block, data, cp.iv)

	return cipherBlock, err
}

func decrypt(b cipher.Block, ciphered []byte, iv string) ([]byte, error) {
	plainSize := len(ciphered)

	if len(ciphered)%aes.BlockSize != 0 {
		return nil, errors.New("Invalid AES cipher encrypted file")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	plain := make([]byte, plainSize)
	mode := cipher.NewCBCDecrypter(b, []byte(iv))
	mode.CryptBlocks(plain, ciphered)

	if len(plain)-int(plain[plainSize-1]) < 0 {
		return nil, errors.New("Invalid AES cipher encrypted file")
	}

	realData := plain[:len(plain)-int(plain[plainSize-1])]

	return realData, nil
}
