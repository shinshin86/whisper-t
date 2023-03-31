package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"time"
)

const (
	transcriptionURL = "https://api.openai.com/v1/audio/transcriptions"
)

type Transcript struct {
	Text string `json:"text"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: The file to be processed must be passed as an argument.")
		fmt.Println("USAGE: OPENAI_API_KEY={your api key} whisper-t {file}")
		os.Exit(1)
	}

	inputFile := os.Args[1]

	// TODO: Since the video will fail if it is too long, it should originally be split into 1 or 2 minute segments and run
	transcript, err := transcribeAudio(inputFile)
	if err != nil {
		fmt.Printf("Error: transcription error: %v\n", err)
	}

	// show result
	fmt.Println(transcript.Text)
}

func transcribeAudio(audioFile string) (*Transcript, error) {
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		fmt.Println("Error: Required OPENAI_API_KEY")
		os.Exit(1)
	}

	file, err := os.Open(audioFile)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to open audio file: %w", err)
	}
	defer file.Close()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(audioFile)))
	partHeader.Set("Content-Type", "videp/mp4")

	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to create form part: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to copy file into form part: %w", err)
	}

	err = writer.WriteField("model", "whisper-1")
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to write model field: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to close writer: %w", err)
	}

	req, err := http.NewRequest("POST", transcriptionURL, &b)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+openaiAPIKey)

	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error: Non-200 response: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to read response body: %w", err)
	}

	var transcript Transcript
	err = json.Unmarshal(body, &transcript)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to unmarshal JSON: %w", err)
	}

	return &transcript, nil
}
