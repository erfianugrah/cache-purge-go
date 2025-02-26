package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Success prints a success message with a checkmark
func Success(message string, args ...interface{}) {
	fmt.Printf("✅ "+message+"\n", args...)
}

// Error prints an error message with a cross
func Error(message string, args ...interface{}) {
	fmt.Printf("❌ "+message+"\n", args...)
}

// Warning prints a warning message
func Warning(message string, args ...interface{}) {
	fmt.Printf("⚠️ "+message+"\n", args...)
}

// Info prints an info message
func Info(message string, args ...interface{}) {
	fmt.Printf("ℹ️ "+message+"\n", args...)
}

// Separator prints a horizontal line
func Separator() {
	fmt.Println(strings.Repeat("-", 80))
}

// Header prints a header with a separator
func Header(title string) {
	fmt.Println("\n" + title)
	Separator()
}

// FormatJSON formats JSON data for display
func FormatJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}
	return string(jsonBytes)
}

// PrettyPrintResults formats operation results
func PrettyPrintResults(success, failure int) {
	fmt.Printf("\nSummary: %d successful, %d failed\n", success, failure)
	if failure > 0 {
		Error("Some operations failed")
	} else {
		Success("All operations completed successfully")
	}
}

// TableHeader prints a formatted table header
func TableHeader(columns []string, widths []int) {
	for i, col := range columns {
		fmt.Printf("%-*s", widths[i], col)
	}
	fmt.Println()

	// Print separator line
	for _, width := range widths {
		fmt.Print(strings.Repeat("-", width))
	}
	fmt.Println()
}

// TableRow prints a formatted table row
func TableRow(values []string, widths []int) {
	for i, val := range values {
		fmt.Printf("%-*s", widths[i], val)
	}
	fmt.Println()
}
