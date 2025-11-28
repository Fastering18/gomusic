package parsers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lrstanley/go-ytdlp"
)

type YtdlpWrapper struct {
	cookiePath string
}

// NewYtdlpWrapper initializes the wrapper and sets up the cookie file from Env
func NewYtdlpWrapper() *YtdlpWrapper {
	cookiePath := ""
	
	// Check for the Environment Variable
	encodedCookies := os.Getenv("YOUTUBE_COOKIES")
	if encodedCookies != "" {
		// Attempt to decode Base64
		decoded, err := base64.StdEncoding.DecodeString(encodedCookies)
		if err != nil {
			log.Println("Warning: Failed to decode YOUTUBE_COOKIES (is it Base64?), trying plain text...")
			decoded = []byte(encodedCookies)
		}

		// Write to a local file
		path := "./cookies.txt"
		err = os.WriteFile(path, decoded, 0644)
		if err != nil {
			log.Printf("Error writing cookies.txt: %v", err)
		} else {
			cookiePath = path
			log.Println("Successfully loaded cookies from Environment Variable")
		}
	}

	return &YtdlpWrapper{
		cookiePath: cookiePath,
	}
}

func (y *YtdlpWrapper) getClient() *ytdlp.Command {
	cmd := ytdlp.New()
	if y.cookiePath != "" {
		cmd = cmd.Cookies(y.cookiePath)
	}
	return cmd
}

func (y *YtdlpWrapper) GetStreamURL(url string) (string, error) {
	// Use helper to ensure cookies are applied
	dl := y.getClient().GetURL()

	result, err := dl.Run(context.TODO(), url)
	if err != nil {
		return "", fmt.Errorf("failed to execute yt-dlp: %w", err)
	}

	lines := strings.Split(result.Stdout, "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no valid URL found in output")
	}

	// Sometimes output has warnings, grab the last valid URL
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "http") {
			return line, nil
		}
	}

	return "", fmt.Errorf("no valid URL found in output")
}

func (y *YtdlpWrapper) GetMetaInfo(url string) (Meta, error) {
	timestamp := time.Now().Format("20060102_150405")
	
	// Use helper to ensure cookies are applied
	dl := y.getClient().DumpJSON().SkipDownload().Output(timestamp + ".%(ext)s")
	
	result, err := dl.Run(context.TODO(), url)
	if err != nil {
		return Meta{}, fmt.Errorf("failed to execute yt-dlp: %w", err)
	}

	var meta Meta
	byteResult := []byte(result.Stdout)

	if err := json.Unmarshal(byteResult, &meta); err != nil {
		return Meta{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	meta.Parser = "yt-dlp"
	return meta, nil
}

func (y *YtdlpWrapper) DownloadStream(url string) (*ytdlp.Result, string, error) {
	cacheDir := "./cache"

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	outputFile := filepath.Join(cacheDir, timestamp)

	// Use helper to ensure cookies are applied
	dl := y.getClient().
		NoPart().
		NoPlaylist().
		NoOverwrites().
		NoKeepVideo().
		Format("bestaudio").
		Output(outputFile)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := dl.Run(ctx, url)
	if err != nil {
		_ = os.Remove(outputFile)
		return nil, "", fmt.Errorf("failed to download stream from URL %q: %w", url, err)
	}

	absPath, err := filepath.Abs(outputFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve absolute path for %q: %w", outputFile, err)
	}

	return result, absPath, nil
}