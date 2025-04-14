package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	http.MaxBytesReader(w, r.Body, 1<<30)
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "couldn't validate JWT", err)
		return
	}

	videoIDVal := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDVal)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error parsing videoId", err)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "video doesn't exist", err)
		return
	}

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "user doesn't own the video", err)
		return
	}

	videoFile, videoFileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading formFile", err)
		return
	}
	defer videoFile.Close()

	mediaType, _, err := mime.ParseMediaType(videoFileHeader.Header.Get("Content-Type"))
	if err != nil || mediaType != "video/mp4" {
		respondWithError(w, http.StatusInternalServerError, "not a mp4 video", err)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, videoFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error writing file", err)
		return
	}

	fastTempFile, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error processing video for fast start", err)
		return
	}

	aspectRatio, err := getViedoAspectRatio(fastTempFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading aspect ration", err)
		return
	}

	prefix := "other/"
	if aspectRatio == "16/9" {
		prefix = "landscape/"
	}
	if aspectRatio == "9/16" {
		prefix = "portrait/"
	}
	processedFile, err := os.Open(fastTempFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error opening processed file", err)
		return
	}
	defer os.Remove(processedFile.Name())
	defer processedFile.Close()
	_, err = processedFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error reading from tempFile", err)
		return
	}

	randBytes := make([]byte, 32)
	_, err = rand.Read(randBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating random string", err)
		return
	}

	hexKey := hex.EncodeToString(randBytes)
	fileKey := fmt.Sprintf("%s%s.mp4", prefix, hexKey)
	params := s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(fileKey),
		Body:        processedFile,
		ContentType: aws.String(mediaType)}

	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("error storing obj: %v", err), err)
		return
	}
	videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, fileKey)
	newVideo := videoData
	newVideo.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(newVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video", err)
		return
	}
	signedVideo, err := cfg.dbVideoToSignedVideo(newVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error signing video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, signedVideo)
}
