package ui

import (
	"fmt"
	"os"
	"strings"
)

const (
	codeReset  = "\033[0m"
	codeBold   = "\033[1m"
	codeDim    = "\033[2m"
	codeRed    = "\033[31m"
	codeGreen  = "\033[32m"
	codeYellow = "\033[33m"
	codeCyan   = "\033[36m"
)

var colored = detectColor()

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func paint(code, text string) string {
	if !colored {
		return text
	}
	return code + text + codeReset
}

func Project(name, path string) {
	fmt.Println()
	fmt.Printf("%s %s %s\n", paint(codeCyan, "▸"), paint(codeBold, name), paint(codeDim, path))
}

func Step(format string, args ...any) {
	fmt.Printf("  %s %s\n", paint(codeCyan, "·"), fmt.Sprintf(format, args...))
}

func Info(format string, args ...any) {
	fmt.Printf("  %s %s\n", paint(codeDim, "-"), fmt.Sprintf(format, args...))
}

func Warn(format string, args ...any) {
	fmt.Printf("  %s %s\n", paint(codeYellow, "!"), fmt.Sprintf(format, args...))
}

func Success(format string, args ...any) {
	fmt.Printf("  %s %s\n", paint(codeGreen, "✓"), fmt.Sprintf(format, args...))
}

func Fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", paint(codeRed, "✗"), fmt.Sprintf(format, args...))
}

func Command(line string) {
	fmt.Printf("    %s\n", paint(codeDim, "$ "+line))
}

func Skipped(line string) {
	fmt.Printf("    %s\n", paint(codeDim, "~ "+line+" (dry-run)"))
}

func Output(text string) {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return
	}
	for _, line := range strings.Split(text, "\n") {
		fmt.Printf("      %s\n", paint(codeDim, line))
	}
}

func Section(title string) {
	fmt.Printf("\n%s\n", paint(codeBold, title))
}

func StatusOK(name, detail string) {
	fmt.Printf("  %s %-20s %s\n", paint(codeGreen, "✓"), name, paint(codeDim, detail))
}

func StatusFail(name, detail string) {
	fmt.Printf("  %s %-20s %s\n", paint(codeRed, "✗"), name, detail)
}
