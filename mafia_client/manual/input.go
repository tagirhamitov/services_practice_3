package manual

import (
	"bufio"
	"fmt"
)

func AskString(message string, scanner *bufio.Scanner) (string, error) {
	fmt.Printf("%v: ", message)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read user response")
	}
	return scanner.Text(), nil
}
