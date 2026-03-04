package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ThumbnailSize defines a thumbnail preset.
type ThumbnailSize struct {
	Name   string
	Width  int
	Height int
}

// DefaultThumbnailSizes defines standard thumbnail presets.
var DefaultThumbnailSizes = []ThumbnailSize{
	{Name: "xs", Width: 100, Height: 100},
	{Name: "sm", Width: 300, Height: 300},
	{Name: "md", Width: 600, Height: 600},
	{Name: "lg", Width: 1200, Height: 1200},
}

// Processor handles image processing (resize, thumbnails).
// Uses ImageMagick's convert command if available, otherwise returns originals.
type Processor struct {
	storage  Storage
	sizes    []ThumbnailSize
	hasConvert bool
}

func NewProcessor(storage Storage, sizes []ThumbnailSize) *Processor {
	if sizes == nil {
		sizes = DefaultThumbnailSizes
	}

	// Check if ImageMagick convert is available
	_, err := exec.LookPath("convert")
	hasConvert := err == nil

	return &Processor{
		storage:    storage,
		sizes:      sizes,
		hasConvert: hasConvert,
	}
}

// GenerateThumbnails creates thumbnails for the given image file.
// Returns a map of size name -> storage path.
func (p *Processor) GenerateThumbnails(ctx context.Context, sourcePath string, mimeType string) (map[string]string, error) {
	if !isImage(mimeType) {
		return nil, nil
	}

	if !p.hasConvert {
		return nil, nil
	}

	thumbnails := make(map[string]string)

	for _, size := range p.sizes {
		thumbPath, err := p.generateThumbnail(ctx, sourcePath, size)
		if err != nil {
			return thumbnails, fmt.Errorf("generating %s thumbnail: %w", size.Name, err)
		}
		thumbnails[size.Name] = thumbPath
	}

	return thumbnails, nil
}

func (p *Processor) generateThumbnail(ctx context.Context, sourcePath string, size ThumbnailSize) (string, error) {
	ext := filepath.Ext(sourcePath)
	thumbFilename := fmt.Sprintf("%s_%s%s",
		strings.TrimSuffix(filepath.Base(sourcePath), ext),
		size.Name,
		ext,
	)

	tmpFile, err := os.CreateTemp("", "thumb-*"+ext)
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Use ImageMagick convert
	cmd := exec.CommandContext(ctx, "convert",
		sourcePath,
		"-resize", fmt.Sprintf("%dx%d>", size.Width, size.Height),
		"-quality", "85",
		tmpPath,
	)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running convert: %w", err)
	}

	f, err := os.Open(tmpPath)
	if err != nil {
		return "", fmt.Errorf("opening thumbnail: %w", err)
	}
	defer f.Close()

	info, _ := f.Stat()
	stored, err := p.storage.Store(ctx, thumbFilename, io.Reader(f), info.Size())
	if err != nil {
		return "", fmt.Errorf("storing thumbnail: %w", err)
	}

	return stored.Path, nil
}

func isImage(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}
