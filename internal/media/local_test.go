package media

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorage_Delete(t *testing.T) {
	basePath := t.TempDir()
	storage := &LocalStorage{basePath: basePath, baseURL: "http://localhost/uploads"}

	t.Run("valid path deletes file", func(t *testing.T) {
		relPath := "2026/03/abcdef12-photo.jpg"
		fullPath := filepath.Join(basePath, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		err := storage.Delete(context.Background(), relPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Fatal("expected file to be deleted")
		}
	})

	t.Run("valid path file not found is not an error", func(t *testing.T) {
		err := storage.Delete(context.Background(), "2026/03/abcdef12-missing.jpg")
		if err != nil {
			t.Fatalf("expected no error for missing file, got %v", err)
		}
	})

	t.Run("directory traversal rejected", func(t *testing.T) {
		err := storage.Delete(context.Background(), "../../etc/passwd")
		if err == nil {
			t.Fatal("expected error for directory traversal")
		}
	})

	t.Run("traversal disguised as valid format rejected", func(t *testing.T) {
		err := storage.Delete(context.Background(), "2026/03/../../../etc/passwd")
		if err == nil {
			t.Fatal("expected error for disguised traversal")
		}
	})

	t.Run("empty path rejected", func(t *testing.T) {
		err := storage.Delete(context.Background(), "")
		if err == nil {
			t.Fatal("expected error for empty path")
		}
	})

	t.Run("path without UUID prefix rejected", func(t *testing.T) {
		err := storage.Delete(context.Background(), "2026/03/photo.jpg")
		if err == nil {
			t.Fatal("expected error for path without UUID prefix")
		}
	})

	t.Run("path with only year rejected", func(t *testing.T) {
		err := storage.Delete(context.Background(), "2026/abcdef12-photo.jpg")
		if err == nil {
			t.Fatal("expected error for incomplete path")
		}
	})
}

func TestValidateMediaPath(t *testing.T) {
	basePath := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid path", "2026/03/abcdef12-photo.jpg", false},
		{"valid path with dots in filename", "2026/01/a1b2c3d4-image.thumb.png", false},
		{"empty path", "", true},
		{"bare filename", "photo.jpg", true},
		{"traversal", "../../etc/passwd", true},
		{"no UUID prefix", "2026/03/photo.jpg", true},
		{"short UUID prefix", "2026/03/abcdef1-photo.jpg", true},
		{"non-hex UUID prefix", "2026/03/ghijklmn-photo.jpg", true},
		{"missing filename after UUID", "2026/03/abcdef12-", true},
		{"backslash traversal", "2026/03/abcdef12-..\\..\\etc\\passwd", false}, // OS-dependent, containment check catches
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateMediaPath(basePath, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMediaPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
