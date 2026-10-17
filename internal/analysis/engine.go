package analysis

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
	"github.com/Gokul-Eswar/Spectre/internal/core"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

// QueryCase runs a question-answer flow over a case by reusing the same
// synthesized context used for full analysis.
//
// The function intentionally falls back to returning raw output when the
// analyzer response is not JSON. This keeps CLI behavior useful even when the
// backend model returns plain text.
func QueryCase(caseID string, model string, question string) (string, error) {
	log.Info().
		Str("case_id", caseID).
		Str("model", model).
		Str("question", question).
		Msg("case_query_started")

	// 1. Fetch Case
	c, err := storage.GetCase(caseID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch case %s: %w", caseID, err)
	}

	// 2. Build Context
	contextData, err := BuildCaseContext(caseID)
	if err != nil {
		return "", fmt.Errorf("failed to build context for case %s: %w", caseID, err)
	}

	// 3. Prepare Request
	req := analyzer.Request{
		Task:     "query",
		CaseID:   caseID,
		CaseName: c.Name,
		Context:  contextData,
		Model:    model,
		Data:     question, // Pass question as data
		LLMConfig: analyzer.GetLLMConfig(),
	}

	// 4. Run Task
	responseJSON, err := analyzer.GlobalTaskRunner.Run(req)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	// 5. Parse
	var resp struct {
		Answer string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		// If it's not JSON, return as is (fallback)
		return responseJSON, nil
	}

	return resp.Answer, nil
}

// AnalyzeImage performs one-shot vision analysis against a local image.
//
// Image bytes are base64-encoded and passed through the same analyzer bridge
// used by text workflows so provider selection, timeout handling, and auditing
// remain consistent with other LLM tasks.
func AnalyzeImage(imagePath string, prompt string, model string) (string, error) {
	log.Info().
		Str("image", imagePath).
		Str("model", model).
		Msg("visual_analysis_started")

	// Read and base64 encode image
	imgData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}
	base64Str := base64.StdEncoding.EncodeToString(imgData)

	req := analyzer.Request{
		Task:    "vision",
		CaseID:  "visual_analysis", // Optional/dummy case ID for vision
		Context: base64Str,         // Pass base64 image here
		Model:   model,
		Data:    prompt, // Pass prompt here
		LLMConfig: analyzer.GetLLMConfig(),
	}

	responseJSON, err := analyzer.GlobalTaskRunner.Run(req)
	if err != nil {
		return "", fmt.Errorf("visual analysis failed: %w", err)
	}

	var resp struct {
		Answer string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(responseJSON), &resp); err != nil {
		return responseJSON, nil // Fallback
	}

	return resp.Answer, nil
}

// AnalyzeCase executes the end-to-end synthesis pipeline for a case.
//
// Flow summary:
//  1. Load case and build normalized prompt context.
//  2. Compute a deterministic context hash and check cached analysis.
//  3. Delegate synthesis to the Python analyzer bridge.
//  4. Parse the structured response and persist it with the context hash.
//
// The context hash is the cache key, so identical case context avoids repeated
// model calls while preserving deterministic traceability.
func AnalyzeCase(caseID string, model string) (*core.Analysis, error) {
	log.Info().
		Str("case_id", caseID).
		Str("model", model).
		Msg("analysis_started")

	// 1. Fetch Case
	c, err := storage.GetCase(caseID)
	if err != nil {
		log.Error().Err(err).Str("case_id", caseID).Msg("failed to fetch case for analysis")
		return nil, fmt.Errorf("failed to fetch case %s: %w", caseID, err)
	}
	if c == nil {
		log.Error().Str("case_id", caseID).Msg("case not found for analysis")
		return nil, fmt.Errorf("case %s not found", caseID)
	}

	// 2. Build Context
	contextData, err := BuildCaseContext(caseID)
	if err != nil {
		log.Error().Err(err).Str("case_id", caseID).Msg("failed to build case context")
		return nil, fmt.Errorf("failed to build context for case %s: %w", caseID, err)
	}

	// Optimization: Check Cache
	hash := sha256.Sum256([]byte(contextData))
	hashStr := hex.EncodeToString(hash[:])

	cached, err := storage.GetAnalysisByHash(caseID, hashStr)
	if err == nil && cached != nil {
		log.Info().Str("case_id", caseID).Msg("analysis_cache_hit")
		return cached, nil
	}

	// 3. Prepare Bridge Request
	req := analyzer.Request{
		Task:     "synthesize",
		CaseID:   caseID,
		CaseName: c.Name,
		Context:  contextData,
		Model:    model,
		LLMConfig: analyzer.GetLLMConfig(),
	}

	if err := req.Validate(); err != nil {
		log.Error().Err(err).Msg("invalid analyzer request")
		return nil, fmt.Errorf("invalid analyzer request: %w", err)
	}

	// 4. Run Python Analyzer
	responseJSON, err := analyzer.GlobalTaskRunner.Run(req)
	if err != nil {
		log.Error().Err(err).Str("case_id", caseID).Msg("python analyzer task failed")
		return nil, fmt.Errorf("python analysis failed: %w", err)
	}

	// 5. Parse
	var result core.Analysis
	if err := json.Unmarshal([]byte(responseJSON), &result); err != nil {
		log.Error().
			Err(err).
			Str("case_id", caseID).
			Str("response", responseJSON).
			Msg("failed to parse AI response")
		return nil, fmt.Errorf("failed to parse AI response: %w\nResponse was: %s", err, responseJSON)
	}

	result.CaseID = caseID
	result.ContextHash = hashStr

	// 6. Save
	if err := storage.SaveAnalysis(&result); err != nil {
		log.Error().Err(err).Str("case_id", caseID).Msg("failed to save analysis result")
		return nil, fmt.Errorf("failed to save analysis for case %s: %w", caseID, err)
	}

	log.Info().
		Str("case_id", caseID).
		Float64("confidence", result.Confidence).
		Msg("analysis_completed")

	return &result, nil
}
