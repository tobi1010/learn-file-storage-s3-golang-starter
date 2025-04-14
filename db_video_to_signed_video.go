package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	fmt.Println("signing")
	fmt.Printf("dbVideo: %v\n", video)
	if video.VideoURL != nil {
		fmt.Println("pointer not nil")
		fields := strings.Split(*video.VideoURL, ",")
		fmt.Printf("fields: %v\n", fields)
		bucket := fields[0]
		key := fields[1]
		url, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*5)
		if err != nil {
			return database.Video{}, err
		}
		video.VideoURL = &url
	} else {
		fmt.Println("pointer nil")
	}
	return video, nil
}
