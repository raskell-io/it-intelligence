//go:build noanthropic

package provider

import "fmt"

// Built with -tags noanthropic (the default Docker image, because compiling the Anthropic SDK needs more
// memory than a small Docker VM has). Build without the tag to use the anthropic provider.
func newAnthropic(Config) (Provider, error) {
	return nil, fmt.Errorf("this itx was built with -tags noanthropic; rebuild without it to use the anthropic provider")
}
