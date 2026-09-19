package widgets

import (
	"regexp"

	"github.com/thebanri/limoni/core/grapheme"
)

// Validator contains common field validation rules.
type Validator struct {
	Required  bool
	MinLength int
	MaxLength int
	Pattern   string
	Message   string
}

// Validate returns an error message for value, or "" if it passes. Lengths
// count characters as a reader sees them: a family emoji or an accented
// letter written with a combining mark is one character, not several code
// points. The default messages are English; set Message to localise.
func (v Validator) Validate(value string) string {
	if v.Required && value == "" {
		if v.Message != "" {
			return v.Message
		}
		return "This field is required."
	}
	if v.MinLength > 0 && grapheme.Count(value) < v.MinLength {
		if v.Message != "" {
			return v.Message
		}
		return "Too short."
	}
	if v.MaxLength > 0 && grapheme.Count(value) > v.MaxLength {
		if v.Message != "" {
			return v.Message
		}
		return "Too long."
	}
	if v.Pattern != "" {
		matched, err := regexp.MatchString(v.Pattern, value)
		if err == nil && !matched {
			if v.Message != "" {
				return v.Message
			}
			return "Invalid format."
		}
	}
	return ""
}

// ValidateFields returns validation errors by field ID.
func ValidateFields(values map[string]string, rules map[string]Validator) map[string]string {
	errors := make(map[string]string)
	for id, rule := range rules {
		if message := rule.Validate(values[id]); message != "" {
			errors[id] = message
		}
	}
	return errors
}
