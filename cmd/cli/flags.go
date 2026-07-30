package main

import "strings"

// parseFlags splits args into positional arguments and "--name value" (or
// boolean "--name") flags. It's a small hand-rolled parser rather than
// flag.FlagSet because each command's flag set differs and this project has
// no other dependency on the stdlib flag package's error-handling/usage
// conventions to stay consistent with.
func parseFlags(args []string) (positional []string, flags map[string]string) {
	flags = make(map[string]string)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			positional = append(positional, arg)
			continue
		}

		name := strings.TrimPrefix(arg, "--")
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			flags[name] = args[i+1]
			i++
		} else {
			flags[name] = "true"
		}
	}

	return positional, flags
}
