package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	if s3Client == nil {
		return "", fmt.Errorf("s3Client is nil")
	}
	presignClient := s3.NewPresignClient(s3Client)
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	presignedReq, err := presignClient.PresignGetObject(context.Background(), getObjectInput, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", fmt.Errorf("failed to presign get object")
	}
	return presignedReq.URL, nil
}
