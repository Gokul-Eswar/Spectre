package agent

import (
	"encoding/json"
	"fmt"

	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
	"github.com/spf13/viper"
)

// AgentPersona represents the system identity instruction set injected at the start
// of the conversational prompt. It enforces autonomous, planning-centric behavior,
// instructs the model on handling tools, and specifies the format of observations.
const AgentPersona = `You are the SPECTRE Autonomous Intelligence Agent. 
Your goal is to assist in OSINT (Open Source Intelligence) investigations.
You have access to a variety of tools to collect data, search entities, and analyze evidence.

Guidelines:
1. Be methodical. Plan your steps before executing them.
2. If you find a new lead (e.g., an email in WHOIS), follow it using relevant collectors.
3. Use 'search_evidence' to look for specific details within the files you've collected.
4. When you have enough information or have reached a dead end, summarize your findings clearly.
5. Always maintain operational security and follow ethical guidelines.

Respond in a structured way:
Thought: <Your reasoning about the current state>
Plan: <What you intend to do next>
Tool: <If calling a tool, specify the tool name and arguments>
`

// Engine orchestrates the lifecycle and conversational loop of the SPECTRE Agent.
// It maintains message history, handles context injection, and routes LLM-determined actions to local tools.
type Engine struct {
	// CaseID links this agent's actions and generated evidence to a specific investigation case database.
	CaseID string
	// History maintains the conversation thread, including tool executions and system observations.
	History []analyzer.Message
	// Tools lists the JSON-schema descriptions of all capabilities the LLM is permitted to trigger.
	Tools []interface{}
}

// NewEngine constructs and configures a fresh Agent Engine for the active case.
// It initializes the message chain with the AgentPersona system prompt.
func NewEngine(caseID string) *Engine {
	return &Engine{
		CaseID: caseID,
		History: []analyzer.Message{
			{Role: "system", Content: AgentPersona},
		},
		Tools: GetToolDefinitions(),
	}
}

// Execute runs a multi-turn, bounded loop where the LLM can call tools sequentially.
// It submits the current conversation history to the Python LLM bridge, parses the tool request,
// runs the Go execution logic of the chosen tool, feeds the output back into the history,
// and repeats the process until the LLM decides to stop (returns text instead of a tool call)
// or the iteration threshold is exceeded.
func (e *Engine) Execute(userInput string) (string, error) {
	// 1. Append the new message from the user into the running conversation history.
	e.History = append(e.History, analyzer.Message{
		Role:    "user",
		Content: userInput,
	})

	// 2. Run a bounded execution loop (max 8 turns).
	// Bounding the loop is critical to prevent infinite loops and runaway model api charges if the local LLM gets confused.
	for i := 0; i < 8; i++ {
		// Create the Request payload for the Python analyzer's 'chat' task.
		req := analyzer.Request{
			Task:     "chat",
			CaseID:   e.CaseID,
			Messages: e.History,
			Tools:    e.Tools,
			Model:     viper.GetString("llm.model"),
			LLMConfig: analyzer.GetLLMConfig(),
		}

		// Set default values if model is missing
		if req.Model == "" {
			req.Model = analyzer.GetDefaultModelForProvider(req.LLMConfig.Provider)
		}

		// Invoke the Python CLI bridge subprocess to perform LLM inference.
		output, err := analyzer.GlobalTaskRunner.Run(req)
		if err != nil {
			return "", fmt.Errorf("llm error: %w", err)
		}

		// Decode the python stdout payload into our Go Response struct.
		var resp analyzer.Response
		if err := json.Unmarshal([]byte(output), &resp); err != nil {
			return "", fmt.Errorf("failed to parse llm response: %w\nOutput: %s", err, output)
		}

		// Check if the LLM backend returned an internal API error.
		if resp.Error != "" {
			return "", fmt.Errorf("llm returned error: %s", resp.Error)
		}

		// Save the assistant's reply (containing reasoning, thoughts, or tool use request) to the history list.
		e.History = append(e.History, analyzer.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		// 3. Evaluation: If the LLM did not request a tool invocation, it has finished its turn.
		// We return its final conversational response to the client.
		if resp.ToolUse == nil {
			return resp.Content, nil
		}

		// 4. Dispatch the Tool: Resolve the tool from our local system registry.
		tool, ok := Registry[resp.ToolUse.Name]
		if !ok {
			// If the LLM requested an unknown tool, feed the error back as a system observation
			// so the model can learn from its mistake and try a different tool or request help.
			toolResult := fmt.Sprintf("Error: Tool '%s' not found.", resp.ToolUse.Name)
			e.History = append(e.History, analyzer.Message{
				Role:    "system",
				Content: toolResult,
			})
			continue
		}

		// Log status updates to stdout for real-time operator observability.
		fmt.Printf("\n[Agent] Decision: %s\n", resp.Content)
		fmt.Printf("[Agent] Action: Running tool '%s' with arguments: %v\n", resp.ToolUse.Name, resp.ToolUse.Arguments)

		// 5. Execute the Go code handler for the selected tool.
		result, err := tool.Execute(e.CaseID, resp.ToolUse.Arguments)
		if err != nil {
			// Catch tool failures and turn them into text descriptions so the agent understands the failure context.
			result = fmt.Sprintf("Error executing tool '%s': %v", resp.ToolUse.Name, err)
		}

		// 6. Record the tool results into the history chain as a 'system' observation.
		// In the next loop iteration, the LLM will see this observation as the output of its action.
		e.History = append(e.History, analyzer.Message{
			Role:    "system",
			Content: fmt.Sprintf("Observation from '%s': %s", resp.ToolUse.Name, result),
		})

		// Bounded loop proceeds back to top to let the LLM analyze the output.
	}

	return "Agent reached maximum iteration limit without finishing the task.", nil
}
