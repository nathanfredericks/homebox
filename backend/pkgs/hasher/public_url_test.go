package hasher

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPublicAttachmentSignature_RoundTrip(t *testing.T) {
	SetAPIKeyPepper([]byte("test-pepper-test-pepper-test-pepper!"))

	id := uuid.New()
	sig := SignPublicAttachmentID(id)

	if sig == "" {
		t.Fatal("expected non-empty signature")
	}
	if !VerifyPublicAttachmentSig(id, sig) {
		t.Error("signature should verify for the same ID")
	}
	if VerifyPublicAttachmentSig(uuid.New(), sig) {
		t.Error("signature must not verify for a different ID")
	}
}

func TestPublicAttachmentSignature_Tampered(t *testing.T) {
	SetAPIKeyPepper([]byte("test-pepper-test-pepper-test-pepper!"))

	id := uuid.New()
	sig := SignPublicAttachmentID(id)

	flip := "A"
	if strings.HasPrefix(sig, "A") {
		flip = "B"
	}
	tampered := flip + sig[1:]
	if VerifyPublicAttachmentSig(id, tampered) {
		t.Error("tampered signature must not verify")
	}

	if VerifyPublicAttachmentSig(id, "") {
		t.Error("empty signature must not verify")
	}
	if VerifyPublicAttachmentSig(id, "%%%not-base64%%%") {
		t.Error("malformed base64 must not verify")
	}
}

func TestPublicAttachmentSignature_StableAcrossCalls(t *testing.T) {
	SetAPIKeyPepper([]byte("test-pepper-test-pepper-test-pepper!"))

	id := uuid.New()
	if SignPublicAttachmentID(id) != SignPublicAttachmentID(id) {
		t.Error("signatures must be deterministic for a fixed pepper and ID")
	}
}

func TestPublicAttachmentSignature_ChangesWithPepper(t *testing.T) {
	id := uuid.New()

	SetAPIKeyPepper([]byte("pepper-one-pepper-one-pepper-one-111"))
	first := SignPublicAttachmentID(id)

	SetAPIKeyPepper([]byte("pepper-two-pepper-two-pepper-two-222"))
	second := SignPublicAttachmentID(id)

	if first == second {
		t.Error("rotating the pepper must invalidate previous signatures")
	}
	if VerifyPublicAttachmentSig(id, first) {
		t.Error("old signature must not verify after pepper rotation")
	}
}
