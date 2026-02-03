package security

import (
	"testing"
	"time"

	"github.com/cychiuae/shhh/internal/crypto"
)

func TestIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "nil expiration - never expires",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "past date - expired",
			expiresAt: timePtr(time.Now().Add(-24 * time.Hour)),
			want:      true,
		},
		{
			name:      "future date - not expired",
			expiresAt: timePtr(time.Now().Add(365 * 24 * time.Hour)),
			want:      false,
		},
		{
			name:      "just expired - expired",
			expiresAt: timePtr(time.Now().Add(-1 * time.Second)),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := crypto.IsExpired(tt.expiresAt)
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExpiringSoon(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		days      int
		want      bool
	}{
		{
			name:      "nil expiration - never expiring",
			expiresAt: nil,
			days:      30,
			want:      false,
		},
		{
			name:      "expires in 10 days with 30 day threshold",
			expiresAt: timePtr(time.Now().Add(10 * 24 * time.Hour)),
			days:      30,
			want:      true,
		},
		{
			name:      "expires in 60 days with 30 day threshold",
			expiresAt: timePtr(time.Now().Add(60 * 24 * time.Hour)),
			days:      30,
			want:      false,
		},
		{
			name:      "already expired",
			expiresAt: timePtr(time.Now().Add(-24 * time.Hour)),
			days:      30,
			want:      true,
		},
		{
			name:      "expires just after threshold",
			expiresAt: timePtr(time.Now().Add(31 * 24 * time.Hour)),
			days:      30,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := crypto.IsExpiringSoon(tt.expiresAt, tt.days)
			if got != tt.want {
				t.Errorf("IsExpiringSoon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyInfoExpiration(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()
	crypto.SetProvider(gpg)

	keyInfo, err := gpg.LookupKey("alice@test.com")
	if err != nil {
		t.Fatalf("failed to lookup key: %v", err)
	}

	if keyInfo.Email != "alice@test.com" {
		t.Errorf("email = %q, want %q", keyInfo.Email, "alice@test.com")
	}

	if keyInfo.KeyID == "" {
		t.Error("KeyID should not be empty")
	}

	if keyInfo.Fingerprint == "" {
		t.Error("Fingerprint should not be empty")
	}
}

func TestKeyLookupNotFound(t *testing.T) {
	gpg, cleanup := setupTestGPG(t)
	defer cleanup()

	_, err := gpg.LookupKey("nonexistent@test.com")
	if err == nil {
		t.Error("should error for nonexistent key")
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
