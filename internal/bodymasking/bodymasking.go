package bodymasking

import (
	"errors"
	"fmt"
	"mime"
	"regexp"

	"github.com/speakeasy-api/speakeasy-go-sdk/internal/utils"
)

var (
	ErrMaskNumberTypeError = errors.New("can't mask a number with a string")
	ErrMaskStringTypeError = errors.New("can't mask a string with a number")
)

const (
	stringFieldMatchRegex = `("%s": *)(".*?[^\\]")( *[, \n\r}]?)`
	numberFieldMatchRegex = `("%s": *)(-?[0-9]+\.?[0-9]*)( *[, \n\r}]?)`
)

func MaskBodyRegex(body, mimeType string, stringMasks map[string]string, numberMasks map[string]string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return "", err
	}

	if mediaType != "application/json" {
		return body, nil
	}

	for field, mask := range stringMasks {
		r, err := regexp.Compile(fmt.Sprintf(stringFieldMatchRegex, regexp.QuoteMeta(field)))
		if err != nil {
			return "", err
		}

		body = utils.ReplaceAllStringSubmatchFunc(r, body, func(matches []string) string {
			return fmt.Sprintf(`%s"%s"%s`, matches[1], mask, matches[3])
		})
	}

	for field, mask := range numberMasks {
		r, err := regexp.Compile(fmt.Sprintf(numberFieldMatchRegex, regexp.QuoteMeta(field)))
		if err != nil {
			return "", err
		}

		body = utils.ReplaceAllStringSubmatchFunc(r, body, func(matches []string) string {
			return fmt.Sprintf("%s%v%s", matches[1], mask, matches[3])
		})
	}

	return body, nil
}
