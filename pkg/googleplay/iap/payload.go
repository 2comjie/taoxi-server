package googleplayIAP

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"errors"

	"github.com/spf13/cast"
)

type Profile struct {
	OrderID   uint64
	ProductID int32
}

func EncryptUid(uid uint64, key []byte) (string, error) {
	return encrypt([]byte(cast.ToString(uid)), key)
}

func DecryptUid(value string, key []byte) (uint64, error) {
	data, err := decrypt(value, key)
	if err != nil {
		return 0, err
	}
	return cast.ToUint64(data), nil
}

func EncryptProfile(profile *Profile, key []byte) (string, error) {
	data := make([]byte, 12)
	binary.BigEndian.PutUint64(data[:8], profile.OrderID)
	binary.BigEndian.PutUint32(data[8:], uint32(profile.ProductID))
	return encrypt(data, key)
}

func DecryptProfile(value string, key []byte) (*Profile, error) {
	data, err := decrypt(value, key)
	if err != nil {
		return nil, err
	}
	if len(data) != 12 {
		return nil, errors.New("Google profile长度错误")
	}
	return &Profile{
		OrderID:   binary.BigEndian.Uint64(data[:8]),
		ProductID: int32(binary.BigEndian.Uint32(data[8:])),
	}, nil
}

func encrypt(data, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	padding := block.BlockSize() - len(data)%block.BlockSize()
	data = append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
	cipher.NewCBCEncrypter(block, key).CryptBlocks(data, data)
	return base64.StdEncoding.EncodeToString(data), nil
}

func decrypt(value string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data)%block.BlockSize() != 0 {
		return nil, errors.New("Google payload长度错误")
	}
	cipher.NewCBCDecrypter(block, key).CryptBlocks(data, data)
	padding := int(data[len(data)-1])
	if padding == 0 || padding > block.BlockSize() || !bytes.Equal(data[len(data)-padding:], bytes.Repeat([]byte{byte(padding)}, padding)) {
		return nil, errors.New("Google payload填充错误")
	}
	return data[:len(data)-padding], nil
}
