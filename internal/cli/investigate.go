package cli

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
	"github.com/Gokul-Eswar/Spectre/internal/collector"
	"github.com/Gokul-Eswar/Spectre/internal/core"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var investigateModelName string

var investigateCmd = &cobra.Command{
	Use:   "investigate [target]",
	Short: "Automated end-to-end investigation (One-Shot)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := ensureCollectorBootstrap(); err != nil {
			return err
		}

		target := args[0]

		if err := storage.InitDB(); err != nil {
			return err
		}

		// 1. Create Case
		caseID := uuid.New().String()
		newCase := core.Case{
			ID:          caseID,
			Name:        fmt.Sprintf("Auto-Investigation: %s", target),
			Description: fmt.Sprintf("Automated investigation triggered for %s at %s", target, time.Now().Format(time.RFC3339)),
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if err := storage.CreateCase(&newCase); err != nil {
			return fmt.Errorf("failed to create case: %w", err)
		}
		fmt.Printf("[+] Created Case: %s (ID: %s)\n", newCase.Name, caseID)

		// 2. Run Collectors (Default set)
		collectors := []string{"dns", "whois", "geo", "ports"}
		fmt.Printf("[*] Running collectors: %v\n", collectors)

		for _, name := range collectors {
			ev, err := collector.Run(name, caseID, target, true, nil)
			if err != nil {
				fmt.Printf("    [!] %s failed: %v\n", name, err)
				continue
			}
			fmt.Printf("    [+] %s: %d items\n", name, len(ev))
			for _, e := range ev {
				storage.CreateEvidence(&e)
				storage.IngestEvidence(&e)
			}
		}

		// 3. Analyze
		if investigateModelName == "" {
			investigateModelName = viper.GetString("llm.model")
		}
		if investigateModelName == "" {
			cfg := analyzer.GetLLMConfig()
			investigateModelName = analyzer.GetDefaultModelForProvider(cfg.Provider)
		}
		fmt.Printf("[*] Running AI Analysis with model '%s'...\n", investigateModelName)
		res, err := analysis.AnalyzeCase(caseID, investigateModelName)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		// 4. Report
		fmt.Println("\n--- INVESTIGATION SUMMARY ---")
		fmt.Printf("Confidence: %.2f\n", res.Confidence)

		fmt.Println("\n[ Findings ]")
		for _, f := range res.Findings {
			fmt.Printf("- %s\n", f)
		}

		fmt.Println("\n[ Risks ]")
		for _, r := range res.Risks {
			fmt.Printf("- %s\n", r)
		}

		fmt.Printf("\nSaved to Case ID: %s\n", caseID)
		return nil
	},
}

func init() {
	investigateCmd.Flags().StringVarP(&investigateModelName, "model", "m", "", "LLM model to use for analysis")
	rootCmd.AddCommand(investigateCmd)
}
