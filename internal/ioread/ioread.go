package ioread

import (
	"fmt"
	"os"
)

func ReadOrContent(content string) (string, error) {
	stat, _ := os.Stat(content)
	if stat != nil {
		if stat.IsDir() {
			return "", fmt.Errorf("%s is a directory", content)
		}
		data, err := os.ReadFile(content)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return content, nil
}
