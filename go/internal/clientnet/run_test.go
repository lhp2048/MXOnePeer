package clientnet

import (
	"context"
	"testing"
	"time"
)

func TestRunRejectsEmptyDir(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := Run(ctx, Options{}); err == nil {
		t.Fatal("expected error")
	}
}
