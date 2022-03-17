package Shared

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	UNCHECKED = iota
	VALID
	BAD
)

type Account struct {
	Combo   string
	Capture []string
}

func (c *Account) AddCaptureStr(name string, data string) {
	c.Capture = append(c.Capture, fmt.Sprintf("%s: %s", name, data))
}

func (c *Account) AddCaptureInt(name string, data int) {
	c.Capture = append(c.Capture, fmt.Sprintf("%s: %s", name, strconv.Itoa(data)))
}

func (c *Account) ToString() string {
	return fmt.Sprintf("%s [ %s ]", c.Combo, strings.Join(c.Capture, " | "))
}

type ComboManager struct {
	ComboList []string
}

func (cm *ComboManager) LoadFromFile(filename string) (int, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cm.ComboList = append(cm.ComboList, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return len(cm.ComboList), nil
}
