package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		filePath,
	)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	var result struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return "", err
	}

	if len(result.Streams) == 0 {
		return "", fmt.Errorf("no video streams found in file: %s", filePath)
	}

	width := result.Streams[0].Width
	height := result.Streams[0].Height

	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("invalid video dimensions: %dx%d", width, height)
	}

	ratio := float64(width) / float64(height)

	switch {
	case ratio > 1.7 && ratio < 1.9:
		return "16:9", nil
	case ratio > 0.55 && ratio < 0.60:
		return "9:16", nil
	default:
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	cmd := exec.Command(
		"ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outputFilePath,
	)

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return outputFilePath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignRequest, err := presignClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", err
	}
	return presignRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	splits := strings.Split(*video.VideoURL, ",")
	bucket, key := splits[0], splits[1]

	signedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour*24)
	if err != nil {
		return video, err
	}

	video.VideoURL = &signedURL
	return video, nil
}
