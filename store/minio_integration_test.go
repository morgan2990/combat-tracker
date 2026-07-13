//go:build integration

package store

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const testMinioBucket = "test-bucket"

// withRealMinio points globalMinio at a real client connected to the
// container started in TestMain (see party_integration_test.go), ensuring
// the test bucket exists, and restores the original (nil) globalMinio
// afterward so tests that rely on "Minio not configured" behavior (e.g.
// TestUploadPDF_NotInitialized) aren't affected.
func withRealMinio(t *testing.T) {
	t.Helper()
	client, err := minio.New(testMinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(testMinioAccessKey, testMinioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("minio.New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	exists, err := client.BucketExists(ctx, testMinioBucket)
	if err != nil {
		t.Fatalf("BucketExists() error = %v", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, testMinioBucket, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("MakeBucket() error = %v", err)
		}
	}

	original := globalMinio
	globalMinio = minioStore{client: client, bucket: testMinioBucket}
	t.Cleanup(func() { globalMinio = original })
}

func TestInitMinio_ConnectsSuccessfully(t *testing.T) {
	original := globalMinio
	t.Cleanup(func() { globalMinio = original })
	globalMinio = minioStore{}

	t.Setenv("MINIO_ENDPOINT", testMinioEndpoint)
	t.Setenv("MINIO_ACCESS_KEY", testMinioAccessKey)
	t.Setenv("MINIO_SECRET_KEY", testMinioSecretKey)
	t.Setenv("MINIO_BUCKET", testMinioBucket)

	InitMinio()

	if globalMinio.client == nil {
		t.Fatal("InitMinio() left globalMinio.client nil, want a connected client")
	}
	if globalMinio.bucket != testMinioBucket {
		t.Errorf("InitMinio() bucket = %q, want %q", globalMinio.bucket, testMinioBucket)
	}
}

func TestInitMinio_MissingConfig_NoOp(t *testing.T) {
	original := globalMinio
	t.Cleanup(func() { globalMinio = original })
	globalMinio = minioStore{}

	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")
	t.Setenv("MINIO_BUCKET", "")

	InitMinio()

	if globalMinio.client != nil {
		t.Error("InitMinio() with no config set should leave globalMinio.client nil")
	}
}

func TestInitMinio_InvalidEndpoint_NoOp(t *testing.T) {
	original := globalMinio
	t.Cleanup(func() { globalMinio = original })
	globalMinio = minioStore{}

	t.Setenv("MINIO_ENDPOINT", "not a valid host!!")
	t.Setenv("MINIO_ACCESS_KEY", "key")
	t.Setenv("MINIO_SECRET_KEY", "secret")
	t.Setenv("MINIO_BUCKET", "bucket")

	InitMinio()

	if globalMinio.client != nil {
		t.Error("InitMinio() with an invalid endpoint should leave globalMinio.client nil")
	}
}

func TestUploadPDF_NotInitialized(t *testing.T) {
	original := globalMinio
	t.Cleanup(func() { globalMinio = original })
	globalMinio = minioStore{}

	err := UploadPDF("goblin", bytes.NewReader([]byte("pdf")), 3)
	if err == nil || err.Error() != "minio not initialized" {
		t.Errorf("UploadPDF() error = %v, want \"minio not initialized\"", err)
	}
}

func TestStreamPDF_NotInitialized(t *testing.T) {
	original := globalMinio
	t.Cleanup(func() { globalMinio = original })
	globalMinio = minioStore{}

	_, err := StreamPDF("goblin")
	if err == nil || err.Error() != "minio not initialized" {
		t.Errorf("StreamPDF() error = %v, want \"minio not initialized\"", err)
	}
}

func TestUploadCustomMonsterPDF_NotInitialized(t *testing.T) {
	original := globalMinio
	t.Cleanup(func() { globalMinio = original })
	globalMinio = minioStore{}

	err := UploadCustomMonsterPDF("cm1", bytes.NewReader([]byte("pdf")), 3)
	if err == nil || err.Error() != "minio not initialized" {
		t.Errorf("UploadCustomMonsterPDF() error = %v, want \"minio not initialized\"", err)
	}
}

func TestStreamCustomMonsterPDF_NotInitialized(t *testing.T) {
	original := globalMinio
	t.Cleanup(func() { globalMinio = original })
	globalMinio = minioStore{}

	_, err := StreamCustomMonsterPDF("cm1")
	if err == nil || err.Error() != "minio not initialized" {
		t.Errorf("StreamCustomMonsterPDF() error = %v, want \"minio not initialized\"", err)
	}
}

func TestUploadAndStreamPDF_RoundTrip(t *testing.T) {
	withRealMinio(t)

	content := []byte("%PDF-1.4 fake goblin statblock")
	if err := UploadPDF("roundtrip-goblin", bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("UploadPDF() error = %v", err)
	}

	rc, err := StreamPDF("roundtrip-goblin")
	if err != nil {
		t.Fatalf("StreamPDF() error = %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("StreamPDF() content = %q, want %q", got, content)
	}
}

// TestStreamPDF_KeyNotFound documents that StreamPDF() itself doesn't error
// for a missing key: minio-go's GetObject builds the Object handle lazily
// without a round trip, so the "not found" error only surfaces on the first
// Read from the returned ReadCloser (which is exactly what api.StreamPDF's
// caller does before treating it as a 404).
func TestStreamPDF_KeyNotFound(t *testing.T) {
	withRealMinio(t)

	rc, err := StreamPDF("no-such-monster")
	if err != nil {
		t.Fatalf("StreamPDF() error = %v, want nil (minio-go defers the not-found error to Read)", err)
	}
	defer rc.Close()

	if _, err := io.ReadAll(rc); err == nil {
		t.Error("io.ReadAll() on a never-uploaded key: error = nil, want an error")
	}
}

func TestUploadAndStreamCustomMonsterPDF_RoundTrip(t *testing.T) {
	withRealMinio(t)

	content := []byte("%PDF-1.4 fake custom bandit statblock")
	if err := UploadCustomMonsterPDF("cm-roundtrip", bytes.NewReader(content), int64(len(content))); err != nil {
		t.Fatalf("UploadCustomMonsterPDF() error = %v", err)
	}

	rc, err := StreamCustomMonsterPDF("cm-roundtrip")
	if err != nil {
		t.Fatalf("StreamCustomMonsterPDF() error = %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("StreamCustomMonsterPDF() content = %q, want %q", got, content)
	}
}

// See TestStreamPDF_KeyNotFound for why the error only surfaces on Read.
func TestStreamCustomMonsterPDF_KeyNotFound(t *testing.T) {
	withRealMinio(t)

	rc, err := StreamCustomMonsterPDF("no-such-custom-monster")
	if err != nil {
		t.Fatalf("StreamCustomMonsterPDF() error = %v, want nil (minio-go defers the not-found error to Read)", err)
	}
	defer rc.Close()

	if _, err := io.ReadAll(rc); err == nil {
		t.Error("io.ReadAll() on a never-uploaded key: error = nil, want an error")
	}
}

func TestUploadPDF_And_UploadCustomMonsterPDF_UseDistinctKeys(t *testing.T) {
	withRealMinio(t)

	sameID := "shared-name"
	officialContent := []byte("official monster PDF")
	customContent := []byte("custom monster PDF")

	if err := UploadPDF(sameID, bytes.NewReader(officialContent), int64(len(officialContent))); err != nil {
		t.Fatalf("UploadPDF() error = %v", err)
	}
	if err := UploadCustomMonsterPDF(sameID, bytes.NewReader(customContent), int64(len(customContent))); err != nil {
		t.Fatalf("UploadCustomMonsterPDF() error = %v", err)
	}

	officialRC, err := StreamPDF(sameID)
	if err != nil {
		t.Fatalf("StreamPDF() error = %v", err)
	}
	defer officialRC.Close()
	officialGot, _ := io.ReadAll(officialRC)
	if !bytes.Equal(officialGot, officialContent) {
		t.Errorf("StreamPDF(%q) = %q, want %q (should not be overwritten by the custom-monster upload)", sameID, officialGot, officialContent)
	}

	customRC, err := StreamCustomMonsterPDF(sameID)
	if err != nil {
		t.Fatalf("StreamCustomMonsterPDF() error = %v", err)
	}
	defer customRC.Close()
	customGot, _ := io.ReadAll(customRC)
	if !bytes.Equal(customGot, customContent) {
		t.Errorf("StreamCustomMonsterPDF(%q) = %q, want %q", sameID, customGot, customContent)
	}
}
