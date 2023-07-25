package aes256

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"log"
)

// Cipher 는 AES 256bit 를 위한 핸들러
type Cipher struct {
	key   string
	iv    string
	block cipher.Block
	mode  cipher.BlockMode
}

// SetKey : 256bit(32bytes) 크기의 키를 셋팅합니다.
func (cp *Cipher) SetKey(key string) error {
	if len(key) != 32 {
		return fmt.Errorf("Key must be set 256 bit")
	}

	cp.key = key
	cp.block, _ = aes.NewCipher([]byte(key))

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
	if cp.mode == nil {
		cp.mode = cipher.NewCBCEncrypter(cp.block, []byte(cp.iv))
	}

	cipherBlock := encrypt(cp.mode, data)

	return cipherBlock, nil
}

func padPKCS7(plainText []byte, blockSize int) []byte {
	padding := blockSize - len(plainText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(plainText, padText...)
}

func trimPKCS5(text []byte) []byte {
	padding := text[len(text)-1]
	return text[:len(text)-int(padding)]
}

func encrypt(mode cipher.BlockMode, plain []byte) []byte {
	paddedPlain := padPKCS7([]byte(plain), mode.BlockSize())

	cipherBytes := make([]byte, len(paddedPlain))
	mode.CryptBlocks(cipherBytes, paddedPlain)

	return cipherBytes
}

// Decode : 복호화!
func (cp *Cipher) Decode(data []byte) ([]byte, error) {
	if cp.mode == nil {
		cp.mode = cipher.NewCBCDecrypter(cp.block, []byte(cp.iv))
	}

	cipherBlock, err := decrypt(cp.mode, data)
	return cipherBlock, err
}

func decrypt(mode cipher.BlockMode, ciphered []byte) ([]byte, error) {
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
	mode.CryptBlocks(plain, ciphered)

	trimmedPlain := trimPKCS5(plain)

	return trimmedPlain, nil
}
