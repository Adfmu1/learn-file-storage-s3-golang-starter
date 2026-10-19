package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
