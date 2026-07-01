package store

import (
	"context"
	"errors"
	"io"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type minioStore struct {
	client *minio.Client
	bucket string
}

var globalMinio minioStore

func InitMinio() {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := os.Getenv("MINIO_BUCKET")
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		log.Printf("minio: MINIO_ENDPOINT/MINIO_ACCESS_KEY/MINIO_SECRET_KEY/MINIO_BUCKET not fully set, skipping init")
		return
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Printf("minio: init failed: %v", err)
		return
	}
	globalMinio = minioStore{client: client, bucket: bucket}
}

func UploadPDF(name string, r io.Reader, size int64) error {
	if globalMinio.client == nil {
		return errors.New("minio not initialized")
	}
	key := "monsters/" + name + ".pdf"
	_, err := globalMinio.client.PutObject(context.Background(), globalMinio.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: "application/pdf",
	})
	return err
}

func StreamPDF(name string) (io.ReadCloser, error) {
	if globalMinio.client == nil {
		return nil, errors.New("minio not initialized")
	}
	key := "monsters/" + name + ".pdf"
	obj, err := globalMinio.client.GetObject(context.Background(), globalMinio.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// UploadCustomMonsterPDF and StreamCustomMonsterPDF key objects by the
// custom monster's id rather than its name, since custom monster names are
// not unique across owners (two DMs' same-named PDFs would otherwise
// overwrite each other under the name-keyed scheme official monsters use).
func UploadCustomMonsterPDF(id string, r io.Reader, size int64) error {
	if globalMinio.client == nil {
		return errors.New("minio not initialized")
	}
	key := "custom-monsters/" + id + ".pdf"
	_, err := globalMinio.client.PutObject(context.Background(), globalMinio.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: "application/pdf",
	})
	return err
}

func StreamCustomMonsterPDF(id string) (io.ReadCloser, error) {
	if globalMinio.client == nil {
		return nil, errors.New("minio not initialized")
	}
	key := "custom-monsters/" + id + ".pdf"
	obj, err := globalMinio.client.GetObject(context.Background(), globalMinio.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}
