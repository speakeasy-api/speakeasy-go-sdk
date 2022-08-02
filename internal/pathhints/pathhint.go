package pathhints

import (
	"fmt"
	"regexp"
	"strings"
)

var varMatcher = regexp.MustCompile(`({(.*?:*.*?)}|:(.+?)\/|:(.*)|\*(.+)|\*)`)

// NormalizePathHint will take a path hint from the various support routers/frameworks and normalize it to the OpenAPI spec.
func NormalizePathHint(pathHint string) string {
	matched := false
	out := replaceAllStringSubmatchFunc(varMatcher, pathHint, func(matches []string) string {
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

func replaceAllStringSubmatchFunc(re *regexp.Regexp, str string, repl func([]string) string) string {
	result := ""
	lastIndex := 0

	for _, v := range re.FindAllSubmatchIndex([]byte(str), -1) {
		groups := []string{}
		for i := 0; i < len(v); i += 2 {
			if v[i] == -1 || v[i+1] == -1 {
				groups = append(groups, "")
			} else {
				groups = append(groups, str[v[i]:v[i+1]])
			}
		}

		result += str[lastIndex:v[0]] + repl(groups)
		lastIndex = v[1]
	}

	return result + str[lastIndex:]
}
