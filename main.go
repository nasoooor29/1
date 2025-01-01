package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	_ "embed"

	"github.com/atotto/clipboard"
)

var (
	TOKEN = ""
	GPT   = "https://api.openai.com/v1/chat/completions"
)

var prefixes map[string]string = map[string]string{
	"f ":   "in bahrain, true or false just give me the one word, ",
	"c ":   "assume this question is talking about bahrain, write short answer",
	"q ":   "write short answer",
	"gem ": "write short answer",
	"no ":  "",
}

func HasOurPrefix(str string) (string, bool) {
	if len(str) < 4 {
		return "", false
	}

	for p := range prefixes {
		if strings.HasPrefix(str, p) {
			return p, true
		}
	}

	return "", false
}

func main() {
	var lastContent string
	log.Println("Starting clipboard monitor. Press Ctrl+C to exit.")
	for {
		time.Sleep(1 * time.Second)
		content, err := clipboard.ReadAll()
		if err != nil {
			log.Printf("Failed to read clipboard: %v\n", err)
			continue
		}
		if content == lastContent {
			continue
		}
		lastContent = content
		userPrefix, hasPrefix := HasOurPrefix(content)
		if !hasPrefix {
			continue
		}
		log.Printf("Clipboard content: %v\n", content)

		prompt := strings.TrimPrefix(content, userPrefix)
		if prompt == "" {
			continue
		}
		log.Printf("Prefix: %v, Prompt: %v\n", prefixes[userPrefix], prompt)
		res := SendToGemeniApi(prefixes[userPrefix], prompt)
		err = clipboard.WriteAll(res + "-")
		if err != nil {
			log.Printf("Failed to read clipboard: %v\n", err)
		}
	}
}

//go:embed content.txt
var s string

func SendToGemeniApi(prefix, prompt string) string {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in f", r)
		}
	}()

	promptBytes, err := json.Marshal(struct {
		Model    string `json:"model"`
		Store    bool   `json:"store"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}{
		Model: "gpt-4o",
		Store: true,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{
				Role:    "user",
				Content: s,
			},
			{
				Role:    "user",
				Content: prefix + "\n" + prompt,
			},
		},
	})
	if err != nil {
		log.Printf("err: %v\n", err)
		return "fuck i couldn't do it"
	}
	prompt = string(promptBytes)

	req, err := http.NewRequest("POST", GPT, strings.NewReader(prompt))
	if err != nil {
		log.Printf("err: %v\n", err)
		return "fuck i couldn't do it"
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+TOKEN)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("err: %v\n", err)
		return "fuck i couldn't do it"
	}
	log.Printf("res.StatusCode: %v\n", res.StatusCode)

	if res.StatusCode != 200 {
		return "fuck i couldn't do it"
	}

	var a GptAnswer
	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("err: %v\n", err)
		return "fuck i couldn't do it"
	}
	err = json.Unmarshal(data, &a)
	if err != nil {
		log.Printf("err: %v\n", err)
		return "fuck i couldn't do it"
	}
	return a.Choices[0].Message.Content
}

type GptAnswer struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string      `json:"role"`
			Content string      `json:"content"`
			Refusal interface{} `json:"refusal"`
		} `json:"message"`
		Logprobs     interface{} `json:"logprobs"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
			AudioTokens  int `json:"audio_tokens"`
		} `json:"prompt_tokens_details"`
		CompletionTokensDetails struct {
			ReasoningTokens          int `json:"reasoning_tokens"`
			AudioTokens              int `json:"audio_tokens"`
			AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
			RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
	SystemFingerprint string `json:"system_fingerprint"`
}
