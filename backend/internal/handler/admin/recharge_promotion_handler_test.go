package admin

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUploadBonusPosterMultipart(t *testing.T) {
	// A real PNG, padded to exercise file size boundaries and disk-backed forms.
	var pngData bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	require.NoError(t, encoder.Encode(&pngData, image.NewNRGBA(image.Rect(0, 0, 700, 700))))
	require.Less(t, pngData.Len(), 2<<20)

	for _, tc := range []struct {
		name        string
		size        int
		contentType string
		field       string
		status      int
		message     string
	}{
		{"2 MiB PNG reaches storage", 2 << 20, "", "file", infraerrors.Code(service.ErrSecretEncryptionKeyNotConfigured), infraerrors.Message(service.ErrSecretEncryptionKeyNotConfigured)},
		{"exactly 5 MiB reaches storage", service.BonusPosterMaxBytes, "", "file", infraerrors.Code(service.ErrSecretEncryptionKeyNotConfigured), infraerrors.Message(service.ErrSecretEncryptionKeyNotConfigured)},
		{"file over limit", service.BonusPosterMaxBytes + 1, "", "file", 400, "image must be at most 5 MB"},
		{"request over limit", service.BonusPosterMaxBytes + (128 << 10), "", "file", 413, "upload request too large"},
		{"JSON content type", 2 << 20, "application/json", "file", 400, "invalid image upload"},
		{"missing boundary", 2 << 20, "multipart/form-data", "file", 400, "invalid image upload"},
		{"wrong boundary", 2 << 20, "multipart/form-data; boundary=incorrect", "file", 400, "invalid image upload"},
		{"missing file", 2 << 20, "", "other", 400, "image required"},
		{"empty file", 0, "", "file", 400, "image is empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.size)
			copy(data, pngData.Bytes())
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile(tc.field, "poster.png")
			require.NoError(t, err)
			_, err = part.Write(data)
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			contentType := tc.contentType
			if contentType == "" {
				contentType = writer.FormDataContentType()
			}
			req := httptest.NewRequest(http.MethodPost, "/posters", &body)
			req.Header.Set("Content-Type", contentType)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = req
			// No database or object store: a valid image must get through parsing
			// and decoding to the explicit storage configuration error.
			h := NewPaymentHandler(&service.PaymentService{}, nil, nil, nil)
			h.UploadBonusPoster(ctx)
			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			require.Contains(t, recorder.Body.String(), tc.message)
			if tc.contentType != "" {
				require.NotContains(t, recorder.Body.String(), "at most 5 MB")
			}
		})
	}
}
