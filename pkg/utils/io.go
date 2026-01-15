package utils

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func ReadLineFromStdin(reader *bufio.Scanner, input *string) error {
	if reader == nil {
		reader = bufio.NewScanner(os.Stdin)
	}
	if reader.Scan() {
		*input = reader.Text()
		return nil
	}
	return reader.Err()
}

// PromptOverwriteNewerKey prompts the user when they're trying to upload a key that's older than the server version
// Returns true if the user wants to proceed with the upload, false otherwise
func PromptOverwriteNewerKey(filename string, localTime time.Time, serverTime time.Time) (bool, error) {
	var answer string
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Warning: The server has a newer version of '%s'\n", filename)
	fmt.Printf("  Local file:  %s\n", localTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Server file: %s\n", serverTime.Format("2006-01-02 15:04:05"))
	fmt.Println("Do you want to overwrite the newer server version with your older local file?")
	fmt.Println("1. Yes, overwrite the server file")
	fmt.Println("2. No, skip this file (recommended)")
	fmt.Print("Please choose an option (will skip by default): ")

	if err := ReadLineFromStdin(scanner, &answer); err != nil {
		return false, err
	}
	fmt.Println()

	return answer == "1", nil
}
