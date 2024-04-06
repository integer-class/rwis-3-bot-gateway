package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const BaseUrl = "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent"

// HarmCategory is the category of harm that the model should block.
type HarmCategory string

const (
	HarmCategoryUnspecified      HarmCategory = "HARM_CATEGORY_UNSPECIFIED"
	HarmCategoryDeregatory       HarmCategory = "HARM_CATEGORY_DEROGATORY"
	HarmCategoryToxicity         HarmCategory = "HARM_CATEGORY_TOXICITY"
	HarmCategoryViolence         HarmCategory = "HARM_CATEGORY_VIOLENCE"
	HarmCategorySexual           HarmCategory = "HARM_CATEGORY_SEXUAL"
	HarmCategoryMedical          HarmCategory = "HARM_CATEGORY_MEDICAL"
	HarmCategoryDangerous        HarmCategory = "HARM_CATEGORY_DANGEROUS"
	HarmCategoryHarassment       HarmCategory = "HARM_CATEGORY_HARASSMENT"
	HarmCategoryHateSpeech       HarmCategory = "HARM_CATEGORY_HATE_SPEECH"
	HarmCategorySexuallyExplicit HarmCategory = "HARM_CATEGORY_SEXUALLY_EXPLICIT"
	HarmCategoryDangerousContent HarmCategory = "HARM_CATEGORY_DANGEROUS_CONTENT"
)

// HarmBlockTreshold is the threshold of harm that the model should block.
type HarmBlockTreshold string

const (
	HarmBlockTresholdUnspecified HarmBlockTreshold = "HARM_BLOCK_THRESHOLD_UNSPECIFIED"
	BlockLowAndAbove             HarmBlockTreshold = "BLOCK_LOW_AND_ABOVE"
	BlockMediumAndAbove          HarmBlockTreshold = "BLOCK_MEDIUM_AND_ABOVE"
	BlockOnlyHigh                HarmBlockTreshold = "BLOCK_ONLY_HIGH"
	BlockNone                    HarmBlockTreshold = "BLOCK_NONE"
)

// SafetySettings is the settings for the safety of the model.
type SafetySettings struct {
	Category  HarmCategory      `json:"category"`
	Threshold HarmBlockTreshold `json:"threshold"`
}

// InlineData is the data that is inline in the request. It contains the mime type and the data in bytes.
type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// Part is the part of the content that is sent to gemini.
type Part struct {
	Text       string      `json:"text"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

// Content is the content that is sent to gemini. It contains the parts and the role.
type Content struct {
	Parts []Part `json:"parts"`
	Role  string `json:"role"`
}

func contentFromText(role, text string) Content {
	return Content{
		Parts: []Part{{Text: text}},
		Role:  role,
	}
}

// GenerationConfig is the configuration for the generation of the content.
type GenerationConfig struct {
	StopSequences   []string `json:"stopSequences,omitempty"`
	Temperature     int      `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	TopP            int      `json:"topP,omitempty"`
	TopK            int      `json:"topK,omitempty"`
}

// SafetyRating is the rating of the safety of the content.
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// Candidate is the candidate that is returned from gemini.
type Candidate struct {
	Content      Content        `json:"content"`
	FinishReason string         `json:"finishReason"`
	Index        int            `json:"index"`
	SafetyRating []SafetyRating `json:"safetyRating"`
}

// PromptFeedback is the feedback for the prompt.
type PromptFeedback struct {
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}

// GeminiRequest is the options for the request to gemini. It contains the contents and safety settings.
type GeminiRequest struct {
	Contents         []Content        `json:"contents"`
	SafetySettings   []SafetySettings `json:"safetySettings"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

// GeminiResponse is the response from gemini. It contains the candidates and the safety rating.
type GeminiResponse struct {
	Candidates     []Candidate    `json:"candidates"`
	PromptFeedback PromptFeedback `json:"promptFeedback"`
}

// fetchGeminiResponse will fetch a response from gemini given a prompt.
// The response is fetched from https://generativelanguage.googleapis.com/v1/{model=models/*}:generateContent
func fetchGeminiResponse(prompt string, contexts []Content) (string, error) {
	contents := []Content{
		contentFromText("user", "Bertindaklah seperti seorang perangkat desa yang baik, ramah, suka membantu, dan selalu menjawab dengan singkat, jelas, dan benar."),
		contentFromText("model", "Baiklah, ada yang bisa saya bantu?"),
		contentFromText("user", "Kalo mau ngurus KTP itu dimana ya?"),
		contentFromText("model", "Bisa ke Kantor Desa, nanti akan dibantu oleh Pak RT atau Pak RW."),
		contentFromText("user", "Terima kasih."),
		contentFromText("model", "Baik. Ada yang bisa saya bantu lagi?"),
	}
	contents = append(contents, contexts...)
	contents = append(contents, contentFromText("user", prompt))

	log.Debug().Msgf("Sending prompt to gemini: %+v", contents)

	requestData := GeminiRequest{
		Contents: contents,
		SafetySettings: []SafetySettings{
			{Category: HarmCategoryHarassment, Threshold: BlockLowAndAbove},
			{Category: HarmCategoryHateSpeech, Threshold: BlockLowAndAbove},
			{Category: HarmCategorySexuallyExplicit, Threshold: BlockLowAndAbove},
			{Category: HarmCategoryDangerousContent, Threshold: BlockLowAndAbove},
		},
		GenerationConfig: GenerationConfig{
			MaxOutputTokens: 512,
		},
	}

	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal json body")
	}

	request, err := http.NewRequest("POST", BaseUrl+"?key="+os.Getenv("GEMINI_API_KEY"), bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", errors.Wrap(err, "failed to construct request to gemini")
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch response from gemini")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("failed to close response body")
		}
	}(response.Body)

	var geminiResponse GeminiResponse
	err = json.NewDecoder(response.Body).Decode(&geminiResponse)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode response from gemini")
	}

	log.Debug().Msgf("Received response from gemini: %+v", geminiResponse)

	if len(geminiResponse.Candidates) == 0 {
		return "", nil
	}

	content := geminiResponse.Candidates[0].Content
	if len(content.Parts) == 0 {
		return "", nil
	}

	return content.Parts[0].Text, nil
}
