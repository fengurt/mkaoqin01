package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

type recognizeResponse struct {
	Transcript string  `json:"transcript"`
	Status     string  `json:"status"`
	Location   string  `json:"location"`
	Reason     string  `json:"reason"`
	OccurredAt string  `json:"occurredAt"`
	Confidence float64 `json:"confidence"`
	ASRMode    string  `json:"asrMode"`
	NLUMode    string  `json:"nluMode"`
}

type extractResult struct {
	Status     string `json:"status"`
	Location   string `json:"location"`
	Reason     string `json:"reason"`
	OccurredAt string `json:"occurredAt"`
}

func main() {
	addr := envOr("VOICE_ADDR", ":8003")
	hsrv := khttp.NewServer(khttp.Address(addr))
	hsrv.HandleFunc("/v1/voice/recognize", handleRecognize)
	hsrv.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"service": "voice-svc", "status": "ok"})
	})

	app := kratos.New(kratos.Name("voice-svc"), kratos.Server(hsrv))
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func handleRecognize(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseMultipartForm(16 << 20); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	audioBytes, contentType, err := readAudioFromForm(request.MultipartForm)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Use an independent timeout context for outbound AI requests.
	// Request-scoped context may be canceled early by server-level timeouts.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	transcript, confidence, asrMode := transcribeWithFallback(ctx, audioBytes, contentType)
	extracted, nluMode := extractWithFallback(ctx, transcript)

	responseBody := recognizeResponse{
		Transcript: transcript,
		Status:     extracted.Status,
		Location:   extracted.Location,
		Reason:     extracted.Reason,
		OccurredAt: extracted.OccurredAt,
		Confidence: confidence,
		ASRMode:    asrMode,
		NLUMode:    nluMode,
	}
	writeJSON(writer, http.StatusOK, responseBody)
}

func readAudioFromForm(form *multipart.Form) ([]byte, string, error) {
	files := form.File["audio"]
	if len(files) == 0 {
		return nil, "", fmt.Errorf("missing form file: audio")
	}
	file, err := files[0].Open()
	if err != nil {
		return nil, "", fmt.Errorf("unable to open audio file")
	}
	defer file.Close()

	bytesData, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read audio bytes")
	}
	contentType := files[0].Header.Get("Content-Type")
	if contentType == "" {
		contentType = "audio/wav"
	}
	return bytesData, contentType, nil
}

func transcribeWithFallback(ctx context.Context, audioBytes []byte, contentType string) (string, float64, string) {
	speechAPIKey := os.Getenv("SPEECH_API_KEY")
	bigModelAttempted := false
	if speechAPIKey != "" {
		bigModelAttempted = true
		transcript, err := callOpenSpeechBigModelASR(ctx, speechAPIKey, audioBytes, contentType)
		if err == nil && strings.TrimSpace(transcript) != "" {
			return transcript, 0.94, "real_openspeech_bigmodel_apikey"
		}
		if err != nil {
			return mockTranscript(), 0.66, "mock_bigmodel_asr_failed_" + sanitizeMode(err.Error())
		}
	}

	speechToken := os.Getenv("SPEECH_TOKEN")
	speechAppID := os.Getenv("SPEECH_APP_ID")
	if speechToken == "" || speechAppID == "" {
		if bigModelAttempted {
			return mockTranscript(), 0.66, "mock_bigmodel_asr_failed"
		}
		return mockTranscript(), 0.73, "mock_missing_speech_credential"
	}

	transcript, err := callOpenSpeechASR(ctx, speechAppID, speechToken, audioBytes, contentType)
	if err != nil || strings.TrimSpace(transcript) == "" {
		return mockTranscript(), 0.65, "mock_asr_failed"
	}
	return transcript, 0.91, "real_openspeech"
}

func sanitizeMode(rawText string) string {
	trimmed := strings.TrimSpace(rawText)
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	trimmed = strings.ReplaceAll(trimmed, "\n", "_")
	trimmed = strings.ReplaceAll(trimmed, "\r", "_")
	if len(trimmed) > 180 {
		return trimmed[:180]
	}
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func callOpenSpeechBigModelASR(ctx context.Context, apiKey string, audioBytes []byte, contentType string) (string, error) {
	requestID := fmt.Sprintf("intervoice-%d", time.Now().UnixNano())
	requestBody := map[string]any{
		"user": map[string]any{
			"uid": "intervoice-demo",
		},
		"audio": map[string]any{
			"data":    base64.StdEncoding.EncodeToString(audioBytes),
			"format":  audioFormatFromContentType(contentType),
			"codec":   "raw",
			"rate":    16000,
			"bits":    16,
			"channel": 1,
		},
		"request": map[string]any{
			"model_name":           "bigmodel",
			"enable_itn":           true,
			"enable_punc":          true,
			"enable_ddc":           false,
			"enable_speaker_info":  false,
			"enable_channel_split": false,
			"show_utterances":      false,
			"vad_segment":          false,
			"sensitive_words_filter": "",
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)

	httpRequest, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://openspeech.bytedance.com/api/v3/auc/bigmodel/submit", bytes.NewReader(bodyBytes))
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("x-api-key", apiKey)
	httpRequest.Header.Set("X-Api-Resource-Id", envOr("SPEECH_RESOURCE_ID", "volc.seedasr.auc"))
	httpRequest.Header.Set("X-Api-Request-Id", requestID)
	httpRequest.Header.Set("X-Api-Sequence", "-1")

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode >= 300 {
		payload, _ := io.ReadAll(httpResponse.Body)
		return "", fmt.Errorf("bigmodel asr status %d: %s", httpResponse.StatusCode, string(payload))
	}
	_, _ = io.ReadAll(httpResponse.Body)

	// BigModel AUC submit returns ack; transcript is fetched from query endpoint.
	return pollOpenSpeechBigModelQuery(ctx, apiKey, requestID)
}

func pollOpenSpeechBigModelQuery(ctx context.Context, apiKey, requestID string) (string, error) {
	queryBody := []byte(`{}`)
	for attempt := 0; attempt < 8; attempt++ {
		httpRequest, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://openspeech.bytedance.com/api/v3/auc/bigmodel/query", bytes.NewReader(queryBody))
		httpRequest.Header.Set("Content-Type", "application/json")
		httpRequest.Header.Set("x-api-key", apiKey)
		httpRequest.Header.Set("X-Api-Resource-Id", envOr("SPEECH_RESOURCE_ID", "volc.seedasr.auc"))
		httpRequest.Header.Set("X-Api-Request-Id", requestID)
		httpRequest.Header.Set("X-Api-Sequence", "-1")

		httpResponse, err := http.DefaultClient.Do(httpRequest)
		if err != nil {
			return "", err
		}
		payload, _ := io.ReadAll(httpResponse.Body)
		httpResponse.Body.Close()
		if httpResponse.StatusCode >= 300 {
			return "", fmt.Errorf("bigmodel query status %d: %s", httpResponse.StatusCode, string(payload))
		}

		var responseData map[string]any
		if err := json.Unmarshal(payload, &responseData); err == nil {
			if transcript, ok := nestedString(responseData, "result", "text"); ok && strings.TrimSpace(transcript) != "" {
				return transcript, nil
			}
			if transcript, ok := nestedString(responseData, "payload_msg", "result", "text"); ok && strings.TrimSpace(transcript) != "" {
				return transcript, nil
			}
			if transcript, ok := responseData["text"].(string); ok && strings.TrimSpace(transcript) != "" {
				return transcript, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return "", fmt.Errorf("bigmodel query transcript unavailable")
}

func callOpenSpeechASR(ctx context.Context, appID, token string, audioBytes []byte, contentType string) (string, error) {
	requestBody := map[string]any{
		"app": map[string]any{
			"appid": appID,
		},
		"user": map[string]any{
			"uid": "intervoice-demo",
		},
		"audio": map[string]any{
			"format":         audioFormatFromContentType(contentType),
			"rate":           16000,
			"language":       "zh-CN",
			"bits":           16,
			"channel":        1,
			"audio_base64":   base64.StdEncoding.EncodeToString(audioBytes),
			"audio_duration": 0,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)

	httpRequest, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://openspeech.bytedance.com/api/v1/auc/submit", bytes.NewReader(bodyBytes))
	httpRequest.Header.Set("Authorization", "Bearer; "+token)
	httpRequest.Header.Set("Resource-Id", "volc.bigasr.auc")
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= 300 {
		return "", fmt.Errorf("asr status %d", httpResponse.StatusCode)
	}
	payload, _ := io.ReadAll(httpResponse.Body)

	var responseData map[string]any
	if err := json.Unmarshal(payload, &responseData); err != nil {
		return "", err
	}
	if transcript, ok := nestedString(responseData, "result", "text"); ok {
		return transcript, nil
	}
	if transcript, ok := responseData["text"].(string); ok {
		return transcript, nil
	}
	return "", fmt.Errorf("asr transcript unavailable")
}

func extractWithFallback(ctx context.Context, transcript string) (extractResult, string) {
	arkAPIKey := os.Getenv("ARK_API_KEY")
	if arkAPIKey == "" {
		return heuristicExtract(transcript), "heuristic_missing_ark_key"
	}
	result, err := callDoubaoNLU(ctx, arkAPIKey, transcript)
	if err != nil || result.Status == "" {
		return heuristicExtract(transcript), "heuristic_nlu_failed"
	}
	return result, "real_ark_chat_completions"
}

func callDoubaoNLU(ctx context.Context, apiKey, transcript string) (extractResult, error) {
	// Support explicit endpoint ID first (recommended for Ark custom deployment),
	// then fallback to ARK_MODEL for default public model names.
	modelName := envOr("ARK_ENDPOINT_ID", "")
	if modelName == "" {
		modelName = envOr("ARK_MODEL", "doubao-seed-1-6-250615")
	}
	systemPrompt := "你是考勤系统语义解析器。只允许输出一个 JSON 对象，不要输出任何解释。JSON 必须包含字段：status/location/reason/occurredAt。status 只能是 CHECK_IN/OFFICE/OUTING/DINING/BUSINESS_TRIP/CHECK_OUT。若无法确定，status 设为 OFFICE，location/reason 可给合理默认值，occurredAt 可留空字符串。"
	userPrompt := fmt.Sprintf("原文：%s\n请严格按如下格式输出：{\"status\":\"OFFICE\",\"location\":\"\",\"reason\":\"\",\"occurredAt\":\"\"}", transcript)

	requestBody := map[string]any{
		"model": modelName,
		"response_format": map[string]any{
			"type": "json_object",
		},
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)

	httpRequest, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://ark.cn-beijing.volces.com/api/v3/chat/completions", bytes.NewReader(bodyBytes))
	httpRequest.Header.Set("Authorization", "Bearer "+apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return extractResult{}, err
	}
	defer httpResponse.Body.Close()
	if httpResponse.StatusCode >= 300 {
		return extractResult{}, fmt.Errorf("nlu status %d", httpResponse.StatusCode)
	}
	payload, _ := io.ReadAll(httpResponse.Body)

	var responseData map[string]any
	if err := json.Unmarshal(payload, &responseData); err != nil {
		return extractResult{}, err
	}
	choices, ok := responseData["choices"].([]any)
	if !ok || len(choices) == 0 {
		return extractResult{}, fmt.Errorf("nlu choices missing")
	}
	firstChoice, _ := choices[0].(map[string]any)
	messageData, _ := firstChoice["message"].(map[string]any)
	content, _ := messageData["content"].(string)
	if content == "" {
		return extractResult{}, fmt.Errorf("nlu content missing")
	}

	var result extractResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return extractResult{}, err
	}
	if result.Status == "" {
		// Secondary recovery path for models that return alternative JSON schema.
		var generic map[string]any
		if err := json.Unmarshal([]byte(content), &generic); err == nil {
			if location, ok := generic["location"].(string); ok {
				result.Location = location
			}
			if reason, ok := generic["reason"].(string); ok {
				result.Reason = reason
			}
			if occurredAt, ok := generic["occurredAt"].(string); ok {
				result.OccurredAt = occurredAt
			}
			if entities, ok := generic["entities"].([]any); ok {
				for _, entity := range entities {
					entityMap, ok := entity.(map[string]any)
					if !ok {
						continue
					}
					entityType, _ := entityMap["type"].(string)
					entityValue, _ := entityMap["value"].(string)
					if result.Location == "" && strings.Contains(entityType, "地点") {
						result.Location = entityValue
					}
				}
			}
		}
	}
	result.Status = normalizeStatus(result.Status)
	if result.Location == "" {
		result.Location = "老区美高梅办公室"
	}
	if result.Reason == "" {
		result.Reason = "工作状态申报"
	}
	if result.OccurredAt == "" {
		result.OccurredAt = time.Now().Format(time.RFC3339)
	}
	return result, nil
}

func heuristicExtract(transcript string) extractResult {
	status := "OFFICE"
	switch {
	case strings.Contains(transcript, "签到"):
		status = "CHECK_IN"
	case strings.Contains(transcript, "签退") || strings.Contains(transcript, "下班"):
		status = "CHECK_OUT"
	case strings.Contains(transcript, "外出") || strings.Contains(transcript, "拜访"):
		status = "OUTING"
	case strings.Contains(transcript, "吃饭") || strings.Contains(transcript, "用餐"):
		status = "DINING"
	case strings.Contains(transcript, "出差"):
		status = "BUSINESS_TRIP"
	case strings.Contains(transcript, "办公室"):
		status = "OFFICE"
	}

	location := "老区美高梅办公室"
	locationPattern := regexp.MustCompile(`(新区|老区)?.{0,8}(办公室|中餐厅|餐厅|酒店|客户现场)`)
	if match := locationPattern.FindString(transcript); match != "" {
		location = strings.TrimSpace(match)
	}

	reason := "工作状态申报"
	if strings.Contains(transcript, "陪") {
		reason = "商务接待"
	} else if strings.Contains(transcript, "拜访") {
		reason = "客户拜访"
	}

	return extractResult{
		Status:     status,
		Location:   location,
		Reason:     reason,
		OccurredAt: time.Now().Format(time.RFC3339),
	}
}

func normalizeStatus(rawStatus string) string {
	upperStatus := strings.ToUpper(strings.TrimSpace(rawStatus))
	allowedStatuses := map[string]struct{}{
		"CHECK_IN":      {},
		"OFFICE":        {},
		"OUTING":        {},
		"DINING":        {},
		"BUSINESS_TRIP": {},
		"CHECK_OUT":     {},
	}
	if _, ok := allowedStatuses[upperStatus]; ok {
		return upperStatus
	}
	return "OFFICE"
}

func mockTranscript() string {
	return "我现在在新区美高梅中餐厅陪王总商务用餐，大概到八点半回办公室"
}

func audioFormatFromContentType(contentType string) string {
	if strings.Contains(contentType, "mpeg") || strings.Contains(contentType, "mp3") {
		return "mp3"
	}
	return "wav"
}

func nestedString(source map[string]any, keys ...string) (string, bool) {
	var current any = source
	for _, key := range keys {
		asMap, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = asMap[key]
		if !ok {
			return "", false
		}
	}
	value, ok := current.(string)
	return value, ok
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
