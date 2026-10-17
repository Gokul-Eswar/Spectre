package cli

import (
	"fmt"
	"strings"

	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Direct interaction with the intelligence layer",
}

var queryCmd = &cobra.Command{
	Use:   "query [question]",
	Short: "Ask a specific question about a case",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		question := strings.Join(args, " ")
		fmt.Printf("Querying case %s about: %s...\n", caseID, question)

		answer, err := analysis.QueryCase(caseID, modelName, question)
		if err != nil {
			return err
		}

		fmt.Printf("\n--- AI RESPONSE ---\n%s\n-------------------\n", answer)
		return nil
	},
}

var visionCmd = &cobra.Command{
	Use:   "vision [image_path] [prompt]",
	Short: "Perform visual analysis on an image using a vision LLM",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("model") {
			if cfgModel := viper.GetString("llm.model"); cfgModel != "" {
				modelName = cfgModel
			}
		}

		imagePath := args[0]
		prompt := "Describe this image in detail. Focus on text, logos, or identifying features."
		if len(args) > 1 {
			prompt = strings.Join(args[1:], " ")
		}

		fmt.Printf("Analyzing image %s with prompt: %s\n", imagePath, prompt)

		answer, err := analysis.AnalyzeImage(imagePath, prompt, modelName)
		if err != nil {
			return err
		}

		fmt.Printf("\n--- VISUAL ANALYSIS ---\n%s\n-----------------------\n", answer)
		return nil
	},
}

func init() {
	queryCmd.Flags().StringVarP(&caseID, "case", "c", "", "Case ID (required)")
	queryCmd.Flags().StringVarP(&modelName, "model", "m", "llama3", "Model to use")

	visionCmd.Flags().StringVarP(&modelName, "model", "m", "llava", "Vision model to use (default: llava)")

	llmCmd.AddCommand(queryCmd)
	llmCmd.AddCommand(visionCmd)
	rootCmd.AddCommand(llmCmd)
}
