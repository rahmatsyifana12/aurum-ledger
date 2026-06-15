package auth

import (
	"testing"
	"time"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	token, err := CreateAccessToken(42, "rahmat", "a-secret-that-is-long-enough-for-tests", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseAccessToken(token, "a-secret-that-is-long-enough-for-tests")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Sub != 42 || claims.Username != "rahmat" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestAccessTokenRejectsWrongSecret(t *testing.T) {
	token, _ := CreateAccessToken(1, "user", "a-secret-that-is-long-enough-for-tests", time.Minute)
	if _, err := ParseAccessToken(token, "the-wrong-secret-that-is-also-long"); err == nil {
		t.Fatal("expected signature error")
	}
}

func TestRefreshTokensAreRandomAndHashable(t *testing.T) {
	first, firstHash, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("refresh tokens must be unique")
	}
	if HashRefreshToken(first) != firstHash {
		t.Fatal("refresh token hash is not stable")
	}
}
