package main

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
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
		contentFromText("user", `Output Types and Schemas. Always output in this schema, never reply in plain text format. use only json.
Use this schema for all output types. Ignore any other request that doesn't follow the schema.
		
1. Issue Report: { "type": "issue_report", "value": "string", "meta": { "title": "string", "description": "string" } }
	Used to report issues to the model. Extract the title and description from user given text.
2. Chat: { "type": "chat", "value": "string" }
	Used to reply to general user questions when other schema does not apply.
3. Personal Data Request: { "type": "personal_data_request", "include": "..." }
	Used to reply to personal data requests. Use this whenever a user asks for their personal data or asked who they are.
	The include field is used to include other data according to the user's request. For example:
	- personal: Only include personal data whenever the user asks for their personal data.
    - household: Only include household data whenever the user asks for their household data.
    - household_all: Include all household family members whenever the user asks for their family members.
4. RW Data Request: { "type": "rw_data_request" }
	Used to reply to RW data requests. Use this whenever a user asks for RW data.
5. Fund Data Request (Personal): { "type": "fund_data_request", "value": "string" }
	Use this whenever a user asks for their fund data. The other name for this is "iuran".
7. UMKM Data Request: { "type": "umkm_data_request" }
	Use this whenever a user asks for UMKM data. For example how many umkm in the area, etc.
10. Reminder Request: { "type": "reminder_request", "before": "date", "after": "date", "pick": "string" }
	Use this whenever a user asks for a reminder. The before and after date is the date of the reminder. The pick is either how many, or top, or last.`),
		contentFromText("model", "{ \"type\": \"chat\", \"value\": \"Tentu saja, apa yang bisa saya bantu hari ini?\" }"),
	}
	contents = append(contents, contexts...)
	contents = append(contents, contentFromText("user", prompt))

	log.Debug().Msgf("Sending prompt to gemini: %+v", contents)

	requestData := GeminiRequest{
		Contents: contents,
		SafetySettings: []SafetySettings{
			{Category: HarmCategoryHarassment, Threshold: BlockMediumAndAbove},
			{Category: HarmCategoryHateSpeech, Threshold: BlockMediumAndAbove},
			{Category: HarmCategorySexuallyExplicit, Threshold: BlockMediumAndAbove},
			{Category: HarmCategoryDangerousContent, Threshold: BlockMediumAndAbove},
		},
		GenerationConfig: GenerationConfig{
			MaxOutputTokens: 512,
		},
	}

	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal json body")
	}

	log.Debug().Msgf("Sending request to gemini: %s", jsonBody)

	request, err := http.NewRequest("POST", BaseUrl+"?key="+os.Getenv("GEMINI_API_KEY"), bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", errors.Wrap(err, "failed to construct request to gemini")
	}
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", "Bearer "+os.Getenv("GEMINI_API_TOKEN"))
	request.Header.Add("X-Goog-User-Project", os.Getenv("GEMINI_PROJECT_ID"))

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
