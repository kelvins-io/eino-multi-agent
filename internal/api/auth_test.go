package api

import "testing"

func TestBearerToken(t *testing.T) {
	if bearerToken("Bearer secret") != "secret" {
		t.Fatal("bearer")
	}
	if bearerToken("secret") != "" {
		t.Fatal("non-bearer should be empty")
	}
}
