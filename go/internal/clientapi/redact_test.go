package clientapi

import "testing"

func TestRedact(t *testing.T) {
	if got := Redact("failed tskey-auth-example now"); got != "failed tskey-auth-*** now" {
		t.Fatalf("%q", got)
	}
}
