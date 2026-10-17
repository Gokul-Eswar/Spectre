package cli

import (
	"fmt"

	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/core"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var modelName string

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Generate an AI-powered intelligence report",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureCollectorBootstrap(); err != nil {
			return err
		}

		if caseID == "" {
			ctxID, err := LoadContext()
			if err == nil && ctxID != "" {
				caseID = ctxID
				fmt.Printf("Using current case: %s\n", caseID)
			}
		}

		if caseID == "" {
			return fmt.Errorf("case ID is required (use --case)")
		}

		if err := storage.InitDB(); err != nil {
			return err
		}

		if !cmd.Flags().Changed("model") {
			if cfgModel := viper.GetString("llm.model"); cfgModel != "" {
				modelName = cfgModel
			}
		}

		fmt.Printf("Analyzing case %s with model %s (via Python)...\n", caseID, modelName)

		result, err := analysis.AnalyzeCase(caseID, modelName)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		printAnalysis(result)
		return nil
	},
}

func printAnalysis(a *core.Analysis) {
	fmt.Printf("\n--- INTELLIGENCE REPORT ---\n")
	fmt.Printf("Confidence: %.2f\n\n", a.Confidence)

	fmt.Println("FINDINGS:")
	for _, f := range a.Findings {
		fmt.Printf("- %s\n", f)
	}
	fmt.Println()

	fmt.Println("RISKS:")
	for _, r := range a.Risks {
		fmt.Printf("- %s\n", r)
	}
	fmt.Println()

	fmt.Println("POTENTIAL CONNECTIONS:")
	for _, c := range a.Connections {
		fmt.Printf("- %s\n", c)
	}
	fmt.Println()

	fmt.Println("RECOMMENDED NEXT STEPS:")
	for _, n := range a.NextSteps {
		fmt.Printf("- %s\n", n)
	}
	fmt.Println()

	if len(a.MissingData) > 0 {
		fmt.Println("MISSING DATA:")
		for _, m := range a.MissingData {
			fmt.Printf("- %s\n", m)
		}
		fmt.Println()
	}

	if len(a.SuggestedCollectors) > 0 {
		fmt.Println("SUGGESTED COLLECTORS:")
		for _, c := range a.SuggestedCollectors {
			fmt.Printf("- %s\n", c)
		}
		fmt.Println()
	}

	fmt.Println("---------------------------")
}

func init() {
	analyzeCmd.Flags().StringVarP(&caseID, "case", "c", "", "Case ID (required)")
	analyzeCmd.Flags().StringVarP(&modelName, "model", "m", "llama3", "AI Model to use (e.g., llama3, mistral)")
	rootCmd.AddCommand(analyzeCmd)
}
