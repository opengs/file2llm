package main

import "github.com/spf13/cobra"

var mainCMD = &cobra.Command{
	Use:   "file2llm",
	Short: "Convert file to LLM text",
	Long:  "Converts files of various formats into plain text suitable for LLMs.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	mainCMD.AddCommand(serveCMD)
}

func main() {
	if err := mainCMD.Execute(); err != nil {
		panic(err)
	}
}
