package robokassa

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type HashAlgorithm string

const (
	HashMD5    HashAlgorithm = "MD5"
	HashSHA256 HashAlgorithm = "SHA256"
)

func buildShpParts(shp map[string]string) []string {
	if len(shp) == 0 {
		return nil
	}
	keys := make([]string, 0, len(shp))
	for k := range shp {
		if strings.HasPrefix(k, "Shp_") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(shp[k])))
	}
	return parts
}

func SignatureForInit(hashAlg HashAlgorithm, merchantLogin, outSum, invID, password1 string, receipt *string, shp map[string]string) (string, error) {
	parts := []string{merchantLogin, outSum, invID}
	if receipt != nil && *receipt != "" {
		parts = append(parts, url.QueryEscape(*receipt))
	}
	parts = append(parts, password1)
	parts = append(parts, buildShpParts(shp)...)
	return hashSignature(hashAlg, strings.Join(parts, ":"))
}

func SignatureForResult(hashAlg HashAlgorithm, outSum, invID, password2 string, shp map[string]string) (string, error) {
	parts := []string{outSum, invID, password2}
	parts = append(parts, buildShpParts(shp)...)
	return hashSignature(hashAlg, strings.Join(parts, ":"))
}

func SignatureForSuccess(hashAlg HashAlgorithm, outSum, invID, password1 string, shp map[string]string) (string, error) {
	parts := []string{outSum, invID, password1}
	parts = append(parts, buildShpParts(shp)...)
	return hashSignature(hashAlg, strings.Join(parts, ":"))
}

func hashSignature(hashAlg HashAlgorithm, payload string) (string, error) {
	switch strings.ToUpper(string(hashAlg)) {
	case string(HashMD5):
		sum := md5.Sum([]byte(payload))
		return hex.EncodeToString(sum[:]), nil
	case string(HashSHA256):
		sum := sha256.Sum256([]byte(payload))
		return hex.EncodeToString(sum[:]), nil
	default:
		return "", errors.New("unsupported hash algorithm")
	}
}

func ConstantTimeEqualSignature(expected, actual string) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	actual = strings.ToLower(strings.TrimSpace(actual))
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
