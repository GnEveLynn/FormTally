package sms

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestTestSenderLogsCodeWhenLoggerIsProvided(t *testing.T) {
	var output bytes.Buffer
	sender := NewTestSender(slog.New(slog.NewTextHandler(&output, nil)))

	if err := sender.Send(context.Background(), "+8613900000001", "login", "123456"); err != nil {
		t.Fatal(err)
	}

	logged := output.String()
	for _, want := range []string{"test SMS code", "phone=+8613900000001", "purpose=login", "code=123456"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output %q does not contain %q", logged, want)
		}
	}
}

func TestTestSenderStaysSilentWithoutLogger(t *testing.T) {
	sender := NewTestSender(nil)

	if err := sender.Send(context.Background(), "+8613900000001", "login", "123456"); err != nil {
		t.Fatal(err)
	}
}
