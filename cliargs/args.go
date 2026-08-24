// Package cliargs provides shared command-line argument helpers.
package cliargs

import "strings"

// ReorderInterspersedFlags makes standard-library flag parsing accept flags
// before or after positional arguments. valueFlags identifies flags that
// consume the following argument when no equals sign is used.
func ReorderInterspersedFlags(args []string, valueFlags ...string) []string {
	consumesValue := make(map[string]struct{}, len(valueFlags))
	for _, name := range valueFlags {
		consumesValue[name] = struct{}{}
	}

	flagArgs := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "--" {
			positionals = append(positionals, args[index+1:]...)
			break
		}
		if argument == "-" || !strings.HasPrefix(argument, "-") {
			positionals = append(positionals, argument)
			continue
		}

		flagArgs = append(flagArgs, argument)
		name := strings.TrimLeft(argument, "-")
		if separator := strings.IndexByte(name, '='); separator >= 0 {
			name = name[:separator]
		}
		if _, consumesNextArgument := consumesValue[name]; consumesNextArgument && !strings.Contains(argument, "=") && index+1 < len(args) {
			index++
			flagArgs = append(flagArgs, args[index])
		}
	}
	return append(flagArgs, positionals...)
}
