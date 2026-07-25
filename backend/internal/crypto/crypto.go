package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/argon2"
)

// SaltLen 持久化盐的字节数
const SaltLen = 16

// encV2Prefix 标记使用 argon2id 强派生密钥的密文；
// 无此前缀的历史密文回退到旧 sha256 派生密钥解密（兼容迁移）。
const encV2Prefix = "v2:"

// Box AES-GCM 加解密。
// gcm 使用当前（argon2id）派生密钥，用于新数据加密；
// legacyGCM 使用旧 sha256 派生密钥，仅用于解密历史密文。
type Box struct {
	gcm       cipher.AEAD
	legacyGCM cipher.AEAD
}

// NewBox 仅用旧 sha256 派生密钥（弱）。保留用于向后兼容与测试；
// 生产部署请使用 NewBoxWithSalt 以启用 argon2id 强派生。
func NewBox(key string) (*Box, error) {
	legacy, err := legacyGCM(key)
	if err != nil {
		return nil, err
	}
	return &Box{gcm: legacy, legacyGCM: legacy}, nil
}

// NewBoxWithSalt 使用 argon2id + 持久化盐派生 32 字节 AES 密钥，
// 抵抗低熵密钥的离线爆破；同时保留旧 sha256 密钥以解密历史密文。
func NewBoxWithSalt(key string, salt []byte) (*Box, error) {
	if len(salt) < SaltLen {
		return nil, fmt.Errorf("salt must be at least %d bytes", SaltLen)
	}
	// argon2id 参数：time=1, memory=64MiB, threads=4；输出 32 字节
	dk := argon2.IDKey([]byte(key), salt, 1, 64*1024, 4, 32)
	block, err := aes.NewCipher(dk)
	if err != nil {
		return nil, err
	}
	strong, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	legacy, err := legacyGCM(key)
	if err != nil {
		return nil, err
	}
	return &Box{gcm: strong, legacyGCM: legacy}, nil
}

// legacyGCM 旧的无盐 sha256 派生密钥
func legacyGCM(key string) (cipher.AEAD, error) {
	sum := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// LoadOrCreateSalt 读取或首次生成持久化盐（0600 权限）。
func LoadOrCreateSalt(path string) ([]byte, error) {
	if raw, err := os.ReadFile(path); err == nil {
		if len(raw) < SaltLen {
			return nil, fmt.Errorf("salt file %s is corrupt (too short)", path)
		}
		return raw, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, salt, 0o600); err != nil {
		return nil, err
	}
	return salt, nil
}

// Encrypt 加密明文，返回 "v2:" + base64(nonce|ciphertext)
func (b *Box) Encrypt(plain string) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := b.gcm.Seal(nonce, nonce, []byte(plain), nil)
	return encV2Prefix + base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt 解密密文：带 "v2:" 前缀走强派生密钥，否则回退旧 sha256 密钥。
func (b *Box) Decrypt(enc string) (string, error) {
	gcm := b.legacyGCM
	if strings.HasPrefix(enc, encV2Prefix) {
		enc = enc[len(encV2Prefix):]
		gcm = b.gcm
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	plain, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// HashToken SHA-256 hex
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomToken 生成 URL 安全随机 token
func RandomToken(nBytes int) (string, error) {
	if nBytes < 16 {
		nBytes = 16
	}
	buf := make([]byte, nBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// MaskURL 脱敏 URL：只保留 scheme://host[:port]，
// 丢弃 userinfo / path / query / fragment，避免订阅 token 或账号密码泄露。
func MaskURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		// 解析失败或不完整：不回显原文
		return "***"
	}
	return fmt.Sprintf("%s://%s/***/***", u.Scheme, u.Host)
}

// MaskToken 脱敏 token（保留首尾各 4 字符）
func MaskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
