package widgets

import "regexp"

// Validator contains common field validation rules.
type Validator struct {
	Required  bool
	MinLength int
	MaxLength int
	Pattern   string
	Message   string
}

func (v Validator) Validate(value string) string {
	if v.Required && value == "" {
		if v.Message != "" {
			return v.Message
		}
		return "Bu alan zorunludur."
	}
	if v.MinLength > 0 && len([]rune(value)) < v.MinLength {
		if v.Message != "" {
			return v.Message
		}
		return "Değer çok kısa."
	}
	if v.MaxLength > 0 && len([]rune(value)) > v.MaxLength {
		if v.Message != "" {
			return v.Message
		}
		return "Değer çok uzun."
	}
	if v.Pattern != "" {
		matched, err := regexp.MatchString(v.Pattern, value)
		if err == nil && !matched {
			if v.Message != "" {
				return v.Message
			}
			return "Geçersiz format."
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
