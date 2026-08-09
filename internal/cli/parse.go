package cli

import "fmt"

// validateFormat checks the --format flag value. An empty string (the
// unset/omitted case) defaults to "markdown", matching Input.Format's
// documented default. Any value other than "", "json", or "markdown" is
// a usage error naming the bad value and the accepted set, per spec
// requirement 3 / 11.
func validateFormat(v string) (string, error) {
	switch v {
	case "", "markdown":
		return "markdown", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("invalid --format value %q: accepted values are \"json\" or \"markdown\"", v)
	}
}

// validateLang checks the --lang flag value, used only by ParseAll
// (cmd/all) -- cmd/java and cmd/typescript never register --lang at all,
// so this is never called on those paths. An empty string is a valid,
// meaningful value: it is the explicit "defer to 003 auto-detection"
// signal (spec requirement 1, constitution Article VI), returned
// unchanged rather than defaulted to anything. Any value other than "",
// "java", or "typescript" is a usage error naming the bad value and the
// accepted set, per spec requirement 1 / 11.
func validateLang(v string) (string, error) {
	switch v {
	case "", "java", "typescript":
		return v, nil
	default:
		return "", fmt.Errorf("invalid --lang value %q: accepted values are \"java\" or \"typescript\" (or omit to defer to auto-detection)", v)
	}
}
