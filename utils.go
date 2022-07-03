package webson

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"io"
)

func magicDigest(challengeKey string, magic []byte) string {
	if magic == nil {
		magic = []byte(DEFAULT_MAGIC_KEY)
	}
	h := sha1.New()
	h.Write([]byte(challengeKey))
	h.Write(magic)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func createChallengeKey() string {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		panic("failed to create challenge key")
	}
	return base64.StdEncoding.EncodeToString(key)
}

func createMask() []byte {
	mask := make([]byte, 4)
	if _, e := rand.Read(mask); e != nil {
		panic("failed to create mask")
	}
	return mask
}

func exceptEOF(e error) error {
	if e == io.EOF {
		return nil
	}
	return e
}
