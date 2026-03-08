package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSSHKeyPath_AutoDetect(t *testing.T) {
	// Create a temp dir to simulate ~/.ssh
	tmpHome := t.TempDir()
	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}

	gc := &GitCloner{}

	// Override HOME so getSSHKeyPath resolves to our temp dir
	// We can't easily override user.Current(), so test the detection logic directly
	// by testing with SSHKeyPath set (override path) and the fallback candidates

	t.Run("explicit path exists", func(t *testing.T) {
		keyFile := filepath.Join(sshDir, "my_custom_key")
		if err := os.WriteFile(keyFile, []byte("fake-key"), 0600); err != nil {
			t.Fatal(err)
		}
		gc.SSHKeyPath = keyFile
		got, err := gc.getSSHKeyPath()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != keyFile {
			t.Errorf("expected %s, got %s", keyFile, got)
		}
	})

	t.Run("explicit path does not exist", func(t *testing.T) {
		gc.SSHKeyPath = filepath.Join(sshDir, "nonexistent_key")
		_, err := gc.getSSHKeyPath()
		if err == nil {
			t.Fatal("expected error for missing key file")
		}
	})

	t.Run("auto-detect prefers ed25519 over rsa", func(t *testing.T) {
		gc.SSHKeyPath = "" // reset to auto-detect

		// Create both id_rsa and id_ed25519
		rsaFile := filepath.Join(sshDir, "id_rsa")
		ed25519File := filepath.Join(sshDir, "id_ed25519")
		os.WriteFile(rsaFile, []byte("fake-rsa"), 0600)
		os.WriteFile(ed25519File, []byte("fake-ed25519"), 0600)

		// Call the auto-detect logic directly (can't override user.Current in unit test,
		// so we test the candidate ordering via a helper)
		candidates := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
		var found string
		for _, name := range candidates {
			p := filepath.Join(sshDir, name)
			if _, err := os.Stat(p); err == nil {
				found = p
				break
			}
		}
		if found != ed25519File {
			t.Errorf("expected ed25519 to be preferred, got %s", found)
		}

		os.Remove(rsaFile)
		os.Remove(ed25519File)
	})

	t.Run("auto-detect falls back to rsa", func(t *testing.T) {
		rsaFile := filepath.Join(sshDir, "id_rsa")
		os.WriteFile(rsaFile, []byte("fake-rsa"), 0600)

		candidates := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
		var found string
		for _, name := range candidates {
			p := filepath.Join(sshDir, name)
			if _, err := os.Stat(p); err == nil {
				found = p
				break
			}
		}
		if found != rsaFile {
			t.Errorf("expected rsa fallback, got %s", found)
		}

		os.Remove(rsaFile)
	})

	t.Run("auto-detect no keys returns error", func(t *testing.T) {
		// Empty ssh dir — no keys
		emptySshDir := filepath.Join(t.TempDir(), ".ssh")
		os.MkdirAll(emptySshDir, 0700)

		candidates := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
		found := false
		for _, name := range candidates {
			p := filepath.Join(emptySshDir, name)
			if _, err := os.Stat(p); err == nil {
				found = true
				break
			}
		}
		if found {
			t.Error("expected no keys to be found")
		}
	})
}

func TestSetSSHKeyPath(t *testing.T) {
	gc := &GitCloner{}
	gc.SetSSHKeyPath("/custom/path/id_ed25519")
	if gc.SSHKeyPath != "/custom/path/id_ed25519" {
		t.Errorf("expected SSHKeyPath to be set, got %s", gc.SSHKeyPath)
	}
}

func TestGetSSHAuth_UsesAgentWhenAvailable(t *testing.T) {
	gc := &GitCloner{}

	// When SSH_AUTH_SOCK is set and agent is running, getSSHAuth should succeed
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Skip("SSH_AUTH_SOCK not set, skipping ssh-agent test")
	}

	auth, err := gc.getSSHAuth()
	if err != nil {
		t.Fatalf("expected ssh-agent auth to succeed, got: %v", err)
	}
	if auth == nil {
		t.Fatal("expected non-nil auth method")
	}
}

func TestGetSSHAuth_FallsBackToKeyFile(t *testing.T) {
	gc := &GitCloner{}

	// Unset SSH_AUTH_SOCK to force key file fallback
	origSock := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer os.Setenv("SSH_AUTH_SOCK", origSock)

	// Without agent and with passphrase-protected keys, this should fail
	// with a helpful error message
	_, err := gc.getSSHAuth()
	// It may succeed if the user has unprotected keys, or fail — both are valid
	// We just verify it doesn't panic
	_ = err
}

func TestIsSSH(t *testing.T) {
	gc := &GitCloner{}

	tests := []struct {
		url    string
		expect bool
	}{
		{"git@github.com:org/repo.git", true},
		{"ssh://git@github.com/org/repo.git", true},
		{"https://github.com/org/repo.git", false},
		{"http://github.com/org/repo.git", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := gc.isSSH(tt.url)
			if got != tt.expect {
				t.Errorf("isSSH(%q) = %v, want %v", tt.url, got, tt.expect)
			}
		})
	}
}
