package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
	"github.com/Gokul-Eswar/Spectre/internal/collector"
	"github.com/Gokul-Eswar/Spectre/internal/core"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
	"github.com/spf13/viper"
)

// Tool defines a capability/capability model that the conversational LLM agent can invoke.
type Tool struct {
	// Name is the unique string identifier for the tool (e.g. "collect").
	Name string `json:"name"`
	// Description explains to the LLM what the tool does and when to invoke it.
	Description string `json:"description"`
	// Parameters specifies the JSON schema representing the arguments the LLM must provide.
	Parameters interface{} `json:"parameters"`
	// Execute defines the Go function handler that is run when the LLM triggers the tool.
	Execute func(caseID string, args map[string]interface{}) (string, error)
}

// Registry stores the global map of all system tools available to the Agent.
// The keys match the Tool.Name field.
var Registry = map[string]Tool{

	// "collect": Runs active or passive data gathering collectors against a target.
	"collect": {
		Name:        "collect",
		Description: "Run a specific collector on a target to gather intelligence. Use 'list_collectors' to see available collectors.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"collector": map[string]interface{}{
					"type":        "string",
					"description": "Name of the collector to run (e.g., 'dns', 'whois').",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "The target to collect data for (domain, IP, email, etc.).",
				},
			},
			"required": []string{"collector", "target"},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			name, ok := args["collector"].(string)
			if !ok {
				return "", fmt.Errorf("collector name is required")
			}
			target, ok := args["target"].(string)
			if !ok {
				return "", fmt.Errorf("target is required")
			}

			// By default, we do not allow active reconnaissance (like active port scanning)
			// via autonomous agent conversations to maintain investigator safety, unless configured.
			evidence, err := collector.RunAndSave(name, caseID, target, false, nil)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("Collector '%s' finished on target '%s'. Generated %d evidence records.", name, target, len(evidence)), nil
		},
	},

	// "list_collectors": Returns all registered collectors.
	// Helps the LLM know which collector plug-in names are available.
	"list_collectors": {
		Name:        "list_collectors",
		Description: "List all available intelligence collectors and their descriptions.",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			collectors := collector.List()
			var sb strings.Builder
			sb.WriteString("Available Collectors:\n")
			for _, c := range collectors {
				sb.WriteString(fmt.Sprintf("- %s: %s (Active: %v)\n", c.Name(), c.Description(), c.IsActive()))
			}
			return sb.String(), nil
		},
	},

	// "search_entities": Performs local database queries looking for specific entities matching a keyword.
	"search_entities": {
		Name:        "search_entities",
		Description: "Search for existing entities in the current case by value or type.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search term (e.g., 'google.com' or 'email').",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			query, ok := args["query"].(string)
			if !ok {
				return "", fmt.Errorf("query is required")
			}

			entities, err := storage.ListEntitiesByCase(caseID)
			if err != nil {
				return "", err
			}

			// Loop and perform case-insensitive substring checks on values and entity types.
			var matched []string
			for _, e := range entities {
				if strings.Contains(strings.ToLower(e.Value), strings.ToLower(query)) ||
					strings.Contains(strings.ToLower(e.Type), strings.ToLower(query)) {
					matched = append(matched, fmt.Sprintf("[%s] %s (Source: %s)", e.Type, e.Value, e.Source))
				}
			}

			if len(matched) == 0 {
				return "No matching entities found in this case.", nil
			}

			return "Found entities:\n" + strings.Join(matched, "\n"), nil
		},
	},

	// "get_case_summary": Provides general statistics on the number of nodes and relations discovered.
	"get_case_summary": {
		Name:        "get_case_summary",
		Description: "Get a high-level summary of the current investigation, including entity counts.",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			c, err := storage.GetCase(caseID)
			if err != nil {
				return "", err
			}
			if c == nil {
				return "Case not found.", nil
			}

			entities, _ := storage.ListEntitiesByCase(caseID)
			rels, _ := storage.ListRelationshipsByCase(caseID)

			return fmt.Sprintf("Case: %s\nDescription: %s\nStats: %d entities, %d relationships.",
				c.Name, c.Description, len(entities), len(rels)), nil
		},
	},

	// "search_evidence": Calls Python vector search to find key matches across all unstructured/structured files.
	"search_evidence": {
		Name:        "search_evidence",
		Description: "Perform a semantic search across all collected evidence files to find specific information.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query (e.g., 'registrant email' or 'vulnerabilities').",
				},
			},
			"required": []string{"query"},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			query, _ := args["query"].(string)

			// Build Request envelope triggering the python side's search_evidence task.
			req := analyzer.Request{
				Task:   "search_evidence",
				CaseID: caseID,
				Data: map[string]interface{}{
					"case_id": caseID,
					"query":   query,
				},
			}

			// Invoke the bridge to call ChromaDB.
			output, err := analyzer.GlobalTaskRunner.Run(req)
			if err != nil {
				return "", err
			}

			var resp struct {
				Results []struct {
					ID      string `json:"id"`
					Content string `json:"content"`
				} `json:"results"`
			}
			if err := json.Unmarshal([]byte(output), &resp); err != nil {
				return "", err
			}

			if len(resp.Results) == 0 {
				return "No relevant information found in evidence files.", nil
			}

			// Format vector similarity output for LLM consumption.
			var sb strings.Builder
			sb.WriteString("Found relevant evidence snippets:\n")
			for _, r := range resp.Results {
				sb.WriteString(fmt.Sprintf("\n--- Source: %s ---\n%s\n", r.ID, r.Content))
			}
			return sb.String(), nil
		},
	},

	// "read_evidence": Reads the raw content of a chosen file in the case storage.
	"read_evidence": {
		Name:        "read_evidence",
		Description: "Read the full content of a specific evidence file.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filename": map[string]interface{}{
					"type":        "string",
					"description": "The name of the file to read (e.g., 'whois_scanme.nmap.org.txt').",
				},
			},
			"required": []string{"filename"},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			filename, _ := args["filename"].(string)

			// Path Traversal Security:
			// Ensure the model doesn't supply path elements like "../" or absolute windows paths to read host system files.
			if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
				return "", fmt.Errorf("invalid filename: path traversal attempt detected")
			}

			filePath := fmt.Sprintf("evidence_storage/%s/%s", caseID, filename)
			data, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to read evidence file '%s': %w", filename, err)
			}

			content := string(data)

			// LLM Window Safeguard:
			// Truncate the file content if it exceeds 15,000 characters to prevent blowing out LLM attention context.
			if len(content) > 15000 {
				content = content[:15000] + "\n\n[... content truncated for brevity ...]"
			}

			return fmt.Sprintf("--- Content of %s ---\n%s", filename, content), nil
		},
	},

	// "analyze_image": Performs visual OCR or screenshot analysis using a multimodal local model.
	"analyze_image": {
		Name:        "analyze_image",
		Description: "Perform visual analysis on an image file (e.g., screenshots) to extract text, logos, or descriptions using a local vision model.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filename": map[string]interface{}{
					"type":        "string",
					"description": "The name of the image file to analyze (must be in evidence_storage/<case_id>/).",
				},
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "Specific instructions or questions about the image (e.g., 'What text is visible?').",
				},
			},
			"required": []string{"filename"},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			filename, _ := args["filename"].(string)
			prompt, ok := args["prompt"].(string)
			if !ok {
				prompt = "Describe this image in detail. Focus on text, logos, or identifying features."
			}

			filePath := fmt.Sprintf("evidence_storage/%s/%s", caseID, filename)
			imgData, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to read image file: %w", err)
			}

			// Encode the image bytes to base64. The Python analyzer endpoint expects a base64 envelope.
			base64Str := base64.StdEncoding.EncodeToString(imgData)

			cfg := analyzer.GetLLMConfig()
			model := viper.GetString("llm.model")
			if model == "" {
				model = analyzer.GetDefaultModelForProvider(cfg.Provider)
			}

			// Prepare request payload for python task "vision".
			req := analyzer.Request{
				Task:      "vision",
				CaseID:    caseID,
				Data:      prompt,
				Context:   base64Str,
				Model:     model,
				LLMConfig: cfg,
			}

			// Invoke the python task runner.
			output, err := analyzer.GlobalTaskRunner.Run(req)
			if err != nil {
				return "", fmt.Errorf("vision analysis failed: %w", err)
			}

			var resp struct {
				Answer string `json:"answer"`
			}
			if err := json.Unmarshal([]byte(output), &resp); err != nil {
				return "", fmt.Errorf("failed to parse vision response: %w", err)
			}

			return resp.Answer, nil
		},
	},

	// "generate_dorks": Uses LLM capabilities to generate OSINT Google search dorks.
	"generate_dorks": {
		Name:        "generate_dorks",
		Description: "Generate a list of specialized Google Dorks for a target domain to find sensitive or leaked information.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{
					"type":        "string",
					"description": "The target domain (e.g., 'example.com').",
				},
			},
			"required": []string{"target"},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			target, _ := args["target"].(string)

			cfg := analyzer.GetLLMConfig()
			model := viper.GetString("llm.model")
			if model == "" {
				model = analyzer.GetDefaultModelForProvider(cfg.Provider)
			}

			req := analyzer.Request{
				Task:      "generate_dorks",
				CaseID:    caseID,
				Data:      target,
				Model:     model,
				LLMConfig: cfg,
			}

			output, err := analyzer.GlobalTaskRunner.Run(req)
			if err != nil {
				return "", fmt.Errorf("dork generation failed: %w", err)
			}

			var resp struct {
				Dorks []string `json:"dorks"`
			}
			if err := json.Unmarshal([]byte(output), &resp); err != nil {
				return "", fmt.Errorf("failed to parse dorks: %w", err)
			}

			if len(resp.Dorks) == 0 {
				return "No dorks generated.", nil
			}

			return "Suggested Google Dorks:\n" + strings.Join(resp.Dorks, "\n"), nil
		},
	},

	// "update_hypotheses": Allows the agent to record its analytical leads/claims into the database.
	"update_hypotheses": {
		Name:        "update_hypotheses",
		Description: "Record or update an intelligence hypothesis/lead for the current case.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"hypothesis": map[string]interface{}{
					"type":        "string",
					"description": "The detailed intelligence hypothesis or lead statement (e.g., 'Target registrant is likely located in Berlin based on whois info').",
				},
				"confidence": map[string]interface{}{
					"type":        "number",
					"description": "Confidence score between 0.0 and 1.0.",
				},
				"evidence_filenames": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
					"description": "List of evidence file names supporting this hypothesis.",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Current status of the lead: 'active', 'verified', or 'refuted'.",
				},
			},
			"required": []string{"hypothesis"},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			hypothesis, ok := args["hypothesis"].(string)
			if !ok {
				return "", fmt.Errorf("hypothesis is required")
			}

			confidence := 0.5
			if confVal, ok := args["confidence"].(float64); ok {
				confidence = confVal
			}

			var filenames []string
			if fnList, ok := args["evidence_filenames"].([]interface{}); ok {
				for _, item := range fnList {
					if fnStr, ok := item.(string); ok {
						filenames = append(filenames, fnStr)
					}
				}
			}

			status := "active"
			if statusVal, ok := args["status"].(string); ok {
				status = statusVal
			}

			// Map human-readable file names (e.g. whois_google.com.txt)
			// to the correct internal DB evidence IDs by scanning existing records.
			var evidenceIDs []string
			if len(filenames) > 0 {
				allEvidence, err := storage.ListEvidenceByCase(caseID)
				if err == nil {
					for _, fn := range filenames {
						found := false
						for _, ev := range allEvidence {
							if strings.HasSuffix(ev.FilePath, fn) || strings.Contains(ev.FilePath, fn) {
								evidenceIDs = append(evidenceIDs, ev.ID)
								found = true
								break
							}
						}
						// If evidence record isn't in DB yet, fallback to raw filename reference.
						if !found {
							evidenceIDs = append(evidenceIDs, fn)
						}
					}
				} else {
					evidenceIDs = filenames
				}
			}

			// Instantiate the intelligence lead model.
			lead := &core.IntelligenceLead{
				CaseID:      caseID,
				Hypothesis:  hypothesis,
				Confidence:  confidence,
				EvidenceIDs: evidenceIDs,
				Status:      status,
			}

			// Save to SQL database.
			if err := storage.CreateLead(lead); err != nil {
				return "", err
			}

			return fmt.Sprintf("Intelligence lead recorded successfully with ID: %s", lead.ID), nil
		},
	},

	// "correlate_personas": Correlates usernames and profiles across platforms into unified persona nodes.
	"correlate_personas": {
		Name:        "correlate_personas",
		Description: "Analyze accounts and usernames in the case to correlate online personas across platforms.",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Execute: func(caseID string, args map[string]interface{}) (string, error) {
			clusters, err := analysis.CorrelatePersonas(caseID)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Correlated %d personas from target entities.", len(clusters)), nil
		},
	},
}

// GetToolDefinitions compiles all tools inside Registry into JSON schemas
// suitable for sending to LLM APIs that support function calling / tool definitions.
func GetToolDefinitions() []interface{} {
	var defs []interface{}
	for _, t := range Registry {
		defs = append(defs, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters,
		})
	}
	return defs
}
