package main

import (
	"bufio"
	"fmt"
	"os"

	"mridu/lang"
)

func main() {
	lang.InitVM()

	args := os.Args[1:]
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "Usage: mridu [script]\n")
		os.Exit(64)
	} else if len(args) == 1 {
		runFile(args[0])
	} else {
		repl()
	}
}

func repl() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			return
		}
		line = line[:len(line)-1]
		lang.Interpret(line)
	}
}

func runFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open file: %s\n", path)
		os.Exit(74)
	}
	result := lang.Interpret(string(data))
	if result == lang.INTERPRET_COMPILE_ERROR {
		os.Exit(65)
	}
	if result == lang.INTERPRET_RUNTIME_ERROR {
		os.Exit(70)
	}
}
