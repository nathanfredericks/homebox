package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"github.com/google/uuid"
)

// publicURLKeyContext domain-separates the public-attachment-URL signing key
// from every other use of the API key pepper. Bump the version suffix if the
// signature format ever changes incompatibly.
const publicURLKeyContext = "homebox/public-attachment-url/v1"

// publicURLKey derives the signing key from the installed pepper. Signatures
// are intentionally non-expiring so URLs stored in external search indexes
// stay valid; rotating the pepper invalidates them all (a reindex restores
// them). Panics when the pepper is missing, same as HashAPIKey.
func publicURLKey() []byte {
	p := apiKeyPepper.Load()
	if p == nil || len(*p) == 0 {
		panic("hasher: API key pepper not configured (call SetAPIKeyPepper at startup)")
	}
	mac := hmac.New(sha256.New, *p)
	mac.Write([]byte(publicURLKeyContext))
	return mac.Sum(nil)
}

// SignPublicAttachmentID returns the stable URL-safe signature authorizing
// unauthenticated access to one attachment.
func SignPublicAttachmentID(id uuid.UUID) string {
	mac := hmac.New(sha256.New, publicURLKey())
	mac.Write(id[:])
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// VerifyPublicAttachmentSig reports whether sig authorizes access to the
// attachment, in constant time.
func VerifyPublicAttachmentSig(id uuid.UUID, sig string) bool {
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, publicURLKey())
	mac.Write(id[:])
	return hmac.Equal(got, mac.Sum(nil))
}
