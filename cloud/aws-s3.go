package cloud

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
)

type S3Conn struct {
	Uploader   *s3manager.Uploader
	Downloader *s3manager.Downloader
	S3         *s3.S3
	Bucket     *string
}

func NewS3Conn(sess *session.Session, bucketName string) *S3Conn {
	var conn S3Conn
	conn.Uploader = s3manager.NewUploader(sess)
	conn.Downloader = s3manager.NewDownloader(sess)
	conn.S3 = s3.New(sess)
	conn.Bucket = aws.String("com.sober-watchdog." + bucketName)
	return &conn
}

func (c *S3Conn) Upload(key string, content io.Reader) error {
	// Create upload input struct
	input := s3manager.UploadInput{
		Bucket: c.Bucket,
		Key:    aws.String(key),
		Body:   content,
	}
	// Upload content to s3
	_, err := c.Uploader.Upload(&input)
	return err
}

func (c *S3Conn) Download(key string) error {

}