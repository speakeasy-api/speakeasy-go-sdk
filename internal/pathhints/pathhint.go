package pathhints

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/utils"
)

var varMatcher = regexp.MustCompile(`({(.*?:*.*?)}|:(.+?)\/|:(.*)|\*(.+)|\*)`)

// NormalizePathHint will take a path hint from the various support routers/frameworks and normalize it to the OpenAPI spec.
func NormalizePathHint(pathHint string) string {
	matched := false
	out := utils.ReplaceAllStringSubmatchFunc(varMatcher, pathHint, func(matches []string) string {
		matched = true

		var varMatch string
		switch {
		case matches[0] == "*":
			varMatch = "wildcard"
		case matches[2] != "":
			varMatch = strings.Split(matches[2], ":")[0]
		case matches[3] != "":
			return fmt.Sprintf("{%s}/", matches[3])
		case matches[4] != "":
			varMatch = matches[4]
		default:
			varMatch = matches[5]
		}

		return fmt.Sprintf("{%s}", varMatch)
	})
	if !matched {
		return pathHint
	}

	return out
}
