package utils

import (
	"bufio"
	"os"
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
