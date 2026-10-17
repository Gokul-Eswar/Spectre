package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/agent"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
	"github.com/spf13/cobra"
)

var chatCaseID string

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive AI investigation session",
	Long: `Open an interactive terminal session with the SPECTRE Agent. 
The agent can run collectors, search entities, and provide summaries of your case using AI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := storage.InitDB(); err != nil {
			return err
		}

		if chatCaseID == "" {
			// List recent cases or ask to create one
			cases, err := storage.ListCases()
			if err != nil || len(cases) == 0 {
				return fmt.Errorf("no cases found. Create one first with 'spectre case create'")
			}
			chatCaseID = cases[0].ID
			fmt.Printf("[*] No case ID provided. Using most recent case: %s (%s)\n", cases[0].Name, chatCaseID)
		} else {
			// Verify case exists
			c, err := storage.GetCase(chatCaseID)
			if err != nil {
				return err
			}
			if c == nil {
				return fmt.Errorf("case with ID '%s' not found", chatCaseID)
			}
			fmt.Printf("[*] Using case: %s (%s)\n", c.Name, chatCaseID)
		}

		engine := agent.NewEngine(chatCaseID)
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("--- SPECTRE AGENT SESSION ---")
		fmt.Println("Type 'exit' or 'quit' to end session.")
		fmt.Println("Type '/help' to open the professional guide.")
		fmt.Println()

		for {
			fmt.Print("You > ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "exit" || input == "quit" {
				break
			}
			if input == "/help" {
				printHelpGuide()
				continue
			}
			if input == "/case" {
				c, err := storage.GetCase(chatCaseID)
				if err != nil || c == nil {
					fmt.Printf("\n[*] Current Case ID: %s (Failed to retrieve details)\n\n", chatCaseID)
				} else {
					fmt.Printf("\n[*] Active Case: %s\n[*] Case ID:     %s\n[*] Status:      %s\n[*] Description: %s\n\n", c.Name, c.ID, c.Status, c.Description)
				}
				continue
			}
			if input == "" {
				continue
			}

			fmt.Print("[*] Agent is thinking...")
			response, err := engine.Execute(input)
			fmt.Print("\r                          \r") // Clear thinking line

			if err != nil {
				fmt.Printf("\n[!] Error: %v\n", err)
				continue
			}

			if response != "" {
				fmt.Printf("\nAgent > %s\n\n", response)
			}
		}

		fmt.Println("Session ended.")
		return nil
	},
}

func printHelpGuide() {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("6")).
		Padding(0, 2).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6")).
		Underline(true).
		MarginTop(1).
		MarginBottom(1)

	cmdStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3"))

	boldStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	fmt.Println()
	fmt.Println(titleStyle.Render("SPECTRE INTERACTIVE HELP GUIDE"))
	fmt.Println("Welcome to the SPECTRE AI assistant. Here is a guide to help you navigate")
	fmt.Println("the platform's capabilities and commands.")

	fmt.Println(headerStyle.Render("1. Conversational Agent Capabilities"))
	fmt.Println("You can ask the agent in plain English to execute tasks. Examples:")
	fmt.Printf("  - %s: %s\n", cmdStyle.Render("Reconnaissance"), "Type \"Scan target.com\" or \"Run dns on target.com\"")
	fmt.Printf("  - %s: %s\n", cmdStyle.Render("Evidence Search"), "Type \"Search for emails in the evidence\"")
	fmt.Printf("  - %s: %s\n", cmdStyle.Render("Summarization"), "Type \"Summarize the current case\" or \"Show entities\"")

	fmt.Println(headerStyle.Render("2. Session Slash Commands"))
	fmt.Printf("  %-15s %s\n", cmdStyle.Render("/help"), "Display this professional guide.")
	fmt.Printf("  %-15s %s\n", cmdStyle.Render("/case"), "Display the active case details (Name, ID, Status, Description).")
	fmt.Printf("  %-15s %s\n", cmdStyle.Render("exit / quit"), "Terminate the current interactive session.")

	fmt.Println(headerStyle.Render("3. SPECTRE OSINT Lifecycle"))
	fmt.Printf("  %-15s %s\n", boldStyle.Render("Case"), "The logical container for an investigation. All target data, evidence, and entity")
	fmt.Printf("                  %s\n", "relations are saved under a unique Case ID.")
	fmt.Printf("  %-15s %s\n", boldStyle.Render("Collect"), "Runs passive or active modules (Geo, Whois, DNS, Ports, GitHub, etc.) to gather data.")
	fmt.Printf("  %-15s %s\n", boldStyle.Render("Analyze"), "Uses configured LLMs (local or cloud) to extract findings, risk rankings, and connections.")
	fmt.Printf("  %-15s %s\n", boldStyle.Render("Visualize"), "Launches a local web server displaying D3.js interactive entity relation graphs.")
	fmt.Printf("  %-15s %s\n", boldStyle.Render("Report"), "Generates structured Markdown summaries or executive-ready branded PDF reports.")

	fmt.Println(headerStyle.Render("4. Operational Security (Ghost Mode)"))
	fmt.Println("  Ensure your proxy (Tor/SOCKS5) configuration is set in configs/default.yaml if you require")
	fmt.Println("  hardened OpSec. If started with --strict flag, all HTTP actions fail-closed if proxy is offline.")
	fmt.Println()
}

func init() {
	chatCmd.Flags().StringVarP(&chatCaseID, "case", "c", "", "Case ID to use for the session")
	rootCmd.AddCommand(chatCmd)
}
