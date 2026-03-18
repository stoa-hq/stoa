package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ---------------------------------------------------------------------------
// Mock service for handler tests
// ---------------------------------------------------------------------------

type mockMediaSvc struct {
	list    func(ctx context.Context, f MediaFilter) ([]Media, int, error)
	upload  func(ctx context.Context, filename, mimeType, altText string, size int64, src io.Reader) (*Media, error)
	getByID func(ctx context.Context, id uuid.UUID) (*Media, error)
	delete  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockMediaSvc) List(ctx context.Context, f MediaFilter) ([]Media, int, error) {
	if m.list != nil {
		return m.list(ctx, f)
	}
	return nil, 0, nil
}
func (m *mockMediaSvc) Upload(ctx context.Context, filename, mimeType, altText string, size int64, src io.Reader) (*Media, error) {
	if m.upload != nil {
		return m.upload(ctx, filename, mimeType, altText, size, src)
	}
	return nil, nil
}
func (m *mockMediaSvc) GetByID(ctx context.Context, id uuid.UUID) (*Media, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, ErrNotFound
}
func (m *mockMediaSvc) Delete(ctx context.Context, id uuid.UUID) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// List — error information disclosure
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Upload — magic byte MIME detection
// ---------------------------------------------------------------------------

// newPNGBytes creates a minimal valid PNG image in memory.
func newPNGBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// createMultipartRequest builds a multipart POST with the given file content and Content-Type header.
func createMultipartRequest(t *testing.T, fileContent []byte, contentType string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="test.png"`)
	h.Set("Content-Type", contentType)
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("Write: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/media", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestHandler_Upload_ValidPNG_Accepted(t *testing.T) {
	uploaded := false
	svc := &mockMediaSvc{
		upload: func(_ context.Context, filename, mimeType, _ string, _ int64, _ io.Reader) (*Media, error) {
			uploaded = true
			if mimeType != "image/png" {
				t.Errorf("expected mime_type image/png, got %s", mimeType)
			}
			return &Media{ID: uuid.New(), Filename: filename, MimeType: mimeType}, nil
		},
	}
	h := NewHandler(svc, zerolog.Nop())

	req := createMultipartRequest(t, newPNGBytes(t), "image/png")
	w := httptest.NewRecorder()
	h.upload(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if !uploaded {
		t.Error("expected svc.Upload to be called")
	}
}

func TestHandler_Upload_SpoofedMIME_Rejected(t *testing.T) {
	svc := &mockMediaSvc{}
	h := NewHandler(svc, zerolog.Nop())

	// Send a plain text file but claim it is image/png.
	req := createMultipartRequest(t, []byte("this is not an image"), "image/png")
	w := httptest.NewRecorder()
	h.upload(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "unsupported_media_type") {
		t.Errorf("expected unsupported_media_type error code, got: %s", body)
	}
}

func TestHandler_Upload_DisallowedType_Rejected(t *testing.T) {
	svc := &mockMediaSvc{}
	h := NewHandler(svc, zerolog.Nop())

	// ELF binary magic bytes.
	elfBytes := []byte{0x7f, 'E', 'L', 'F', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	req := createMultipartRequest(t, elfBytes, "application/x-executable")
	w := httptest.NewRecorder()
	h.upload(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Upload_DetectedMIME_OverridesClientHeader(t *testing.T) {
	svc := &mockMediaSvc{
		upload: func(_ context.Context, _, mimeType, _ string, _ int64, _ io.Reader) (*Media, error) {
			if mimeType != "image/png" {
				t.Errorf("expected detected mime_type image/png, got %s", mimeType)
			}
			return &Media{ID: uuid.New(), MimeType: mimeType}, nil
		},
	}
	h := NewHandler(svc, zerolog.Nop())

	// Send a real PNG but claim it is application/pdf.
	req := createMultipartRequest(t, newPNGBytes(t), "application/pdf")
	w := httptest.NewRecorder()
	h.upload(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// List — error information disclosure
// ---------------------------------------------------------------------------

func TestHandler_List_ServiceError_NoInfoDisclosure(t *testing.T) {
	svc := &mockMediaSvc{
		list: func(_ context.Context, _ MediaFilter) ([]Media, int, error) {
			return nil, 0, errors.New("pq: relation \"media\" does not exist")
		},
	}
	h := NewHandler(svc, zerolog.Nop())

	req := httptest.NewRequest(http.MethodGet, "/media", nil)
	w := httptest.NewRecorder()
	h.list(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "an unexpected error occurred") {
		t.Errorf("expected generic error message in response body, got: %s", body)
	}
	if strings.Contains(body, "pq:") {
		t.Errorf("response body must not contain internal error details, got: %s", body)
	}
}
