package utils

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/a-h/templ"
)

func TemplString(t templ.Component) (string, error) {
	var b bytes.Buffer
	if err := t.Render(context.Background(), &b); err != nil {
		return "", err
	}
	return b.String(), nil
}

func ReplaceStyles(text string, bottom, left []int) string {
	re := regexp.MustCompile(`style="bottom:\s*[\d.]+px;\s*left:\s*[\d.]+px"`)

	replacements := []string{}

	for i := range bottom {
		replacements = append(replacements, fmt.Sprintf(`style="bottom: %vpx; left: %vpx"`, bottom[i], left[i]))
	}

	matches := re.FindAllStringIndex(text, -1)

	var builder strings.Builder
	lastIndex := 0

	for i, match := range matches {
		start, end := match[0], match[1]

		builder.WriteString(text[lastIndex:start])

		if i < len(replacements) {
			builder.WriteString(replacements[i])
		} else {
			builder.WriteString(text[start:end])
		}

		lastIndex = end
	}

	builder.WriteString(text[lastIndex:])

	output := builder.String()

	return output
}

func FormatTime(seconds int) string {
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}
