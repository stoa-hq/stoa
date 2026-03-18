package main

import (
	"bytes"
	"testing"
)

func TestAdminCreate_MissingEmail(t *testing.T) {
	cmd := adminCmd()
	cmd.SetArgs([]string{"create", "--password", "secret123"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --email is missing")
	}
	if err.Error() != "--email is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminCreate_EmptyPassword(t *testing.T) {
	cmd := adminCmd()
	cmd.SetArgs([]string{"create", "--email", "admin@test.com", "--password", ""})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for empty --password")
	}
	if err.Error() != "--password cannot be empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdminCreate_PasswordFlagShowsWarning(t *testing.T) {
	cmd := adminCmd()
	errBuf := &bytes.Buffer{}
	cmd.SetArgs([]string{"create", "--email", "admin@test.com", "--password", "secret123"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(errBuf)

	// Will fail on DB connect, but we can check the warning was emitted before that
	_ = cmd.Execute()

	stderr := errBuf.String()
	if !bytes.Contains([]byte(stderr), []byte("insecure and visible in process listings")) {
		t.Fatalf("expected deprecation warning in stderr, got: %s", stderr)
	}
}

func TestAdminCreate_NoPasswordFlagNonInteractive(t *testing.T) {
	cmd := adminCmd()
	cmd.SetArgs([]string{"create", "--email", "admin@test.com"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no TTY and no --password")
	}
	if err.Error() != "stdin is not a terminal; use --password flag or run interactively" {
		t.Fatalf("unexpected error: %v", err)
	}
}
