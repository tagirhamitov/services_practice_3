package parameters

import (
	"bufio"
	"fmt"
	"os"
)

type ClientParameters struct {
	Username      string
	ServerAddress string
}

func GetClientParameters() (ClientParameters, error) {
	scanner := bufio.NewScanner(os.Stdin)
	params := ClientParameters{}

	username, err := askString("Введите имя пользователя", scanner)
	if err != nil {
		return params, fmt.Errorf("failed to read username: %w", err)
	}
	params.Username = username

	serverAddress, err := askString("Введите адрес сервера", scanner)
	if err != nil {
		return params, fmt.Errorf("failed to read server address: %w", err)
	}
	params.ServerAddress = serverAddress

	return params, nil
}

func askString(message string, scanner *bufio.Scanner) (string, error) {
	fmt.Printf("%v: ", message)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read user response")
	}
	return scanner.Text(), nil
}
