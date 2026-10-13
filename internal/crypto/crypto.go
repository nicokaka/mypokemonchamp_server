package crypto

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func MD5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func SHA1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func GZipEncode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GZipDecode(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	pad := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, pad...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("invalid padded data length %d", len(data))
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, fmt.Errorf("invalid pkcs7 pad byte %d", padLen)
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, fmt.Errorf("invalid pkcs7 padding")
		}
	}
	return data[:len(data)-padLen], nil
}

func AesEncrypt(plain, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plain, aes.BlockSize)
	out := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out, padded)
	return out, nil
}

func AesDecrypt(cipherText, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(cipherText) == 0 || len(cipherText)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not multiple of block size")
	}
	out := make([]byte, len(cipherText))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(out, cipherText)
	return pkcs7Unpad(out, aes.BlockSize)
}

func deriveAuthKeyIV(dummyValue string, win uint32) ([]byte, []byte, error) {
	md5Dummy := MD5Hex(dummyValue)
	h2 := MD5Hex(fmt.Sprintf("%d$%s", win, md5Dummy))
	combined := md5Dummy + h2
	shift := int(win & 31)
	if shift > len(combined) {
		shift = 0
	}
	keyHex := combined[shift:] + combined[:shift]
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, nil, err
	}
	ivHex := MD5Hex(fmt.Sprintf("%s;%d", dummyValue, win))
	ivBytes, err := hex.DecodeString(ivHex)
	if err != nil {
		return nil, nil, err
	}
	return keyBytes, ivBytes, nil
}

// EncryptPmpAuth replicates Net.Encrypt::EncryptRequestPmpAuth / DecryptResponsePmcAuth.
func EncryptPmpAuth(pmpJSON, dummyValue string, serverUnixTime int64) (string, error) {
	win := uint32(uint64(serverUnixTime) / 600)
	key, iv, err := deriveAuthKeyIV(dummyValue, win)
	if err != nil {
		return "", err
	}
	gz, err := GZipEncode([]byte(pmpJSON))
	if err != nil {
		return "", err
	}
	enc, err := AesEncrypt(gz, key, iv)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(enc), nil
}

// DecryptPmcAuth decrypts an incoming Auth pmc payload, trying current time window and 0 (initial request before server time sync).
func DecryptPmcAuth(pmcBase64, dummyValue string) (string, error) {
	trimmed := strings.TrimSpace(pmcBase64)
	if strings.HasPrefix(trimmed, "{") {
		return trimmed, nil
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return "", err
	}
	nowWin := uint32(uint64(time.Now().Unix()) / 600)
	candidates := []uint32{nowWin, 0, nowWin - 1, nowWin + 1}
	for _, win := range candidates {
		key, iv, err := deriveAuthKeyIV(dummyValue, win)
		if err != nil {
			continue
		}
		dec, err := AesDecrypt(raw, key, iv)
		if err != nil {
			continue
		}
		unz, err := GZipDecode(dec)
		if err == nil {
			return string(unz), nil
		}
	}
	return "", fmt.Errorf("failed to decrypt Auth pmc")
}

func deriveApiKeyIV(dummyValue, csrfToken, sessionId string, udVer int) ([]byte, []byte, error) {
	h1 := MD5Hex(fmt.Sprintf("%s$%s$%s", sessionId, csrfToken, dummyValue))
	dummyLong, _ := strconv.ParseInt(dummyValue, 10, 64)
	shift := int(dummyLong & 31)
	if dummyLong < 0 {
		shift = -int((-dummyLong) & 31)
	}
	if shift < 0 || shift > len(h1) {
		shift = 0
	}
	rotated := h1[shift:] + h1[:shift]
	h2 := MD5Hex(fmt.Sprintf("%d%s", udVer, rotated))
	keyBytes, err := hex.DecodeString(h1 + h2)
	if err != nil {
		return nil, nil, err
	}
	ivHex := MD5Hex(fmt.Sprintf("%s=%s=%s", sessionId, dummyValue, csrfToken))
	ivBytes, err := hex.DecodeString(ivHex)
	if err != nil {
		return nil, nil, err
	}
	return keyBytes, ivBytes, nil
}

// EncryptPmpApi replicates Net.Encrypt::EncryptRequestPmpApi / DecryptResponsePmcApi.
func EncryptPmpApi(pmpJSON, dummyValue, csrfToken, sessionId string, udVer int) (string, error) {
	key, iv, err := deriveApiKeyIV(dummyValue, csrfToken, sessionId, udVer)
	if err != nil {
		return "", err
	}
	gz, err := GZipEncode([]byte(pmpJSON))
	if err != nil {
		return "", err
	}
	enc, err := AesEncrypt(gz, key, iv)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(enc), nil
}

// DecryptPmcApi decrypts an incoming API pmc payload, trying candidate udVer values.
func DecryptPmcApi(pmcBase64, dummyValue, csrfToken, sessionId string, udVers ...int) (string, error) {
	trimmed := strings.TrimSpace(pmcBase64)
	if strings.HasPrefix(trimmed, "{") {
		return trimmed, nil
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return "", err
	}
	candidates := append([]int(nil), udVers...)
	candidates = append(candidates, 0, 1, 2, 3, 4, 5)
	for _, uv := range candidates {
		key, iv, err := deriveApiKeyIV(dummyValue, csrfToken, sessionId, uv)
		if err != nil {
			continue
		}
		dec, err := AesDecrypt(raw, key, iv)
		if err != nil {
			continue
		}
		unz, err := GZipDecode(dec)
		if err == nil {
			return string(unz), nil
		}
	}
	return "", fmt.Errorf("failed to decrypt API pmc")
}
