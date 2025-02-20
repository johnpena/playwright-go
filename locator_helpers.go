package playwright

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func convertRegexp(reg *regexp.Regexp) (pattern, flags string) {
	matches := regexp.MustCompile(`\(\?([imsU]+)\)(.+)`).FindStringSubmatch(reg.String())

	if len(matches) == 3 {
		pattern = matches[2]
		flags = matches[1]
	} else {
		pattern = reg.String()
	}
	return
}

func escapeForAttributeSelector(value string, exact bool) string {
	suffix := "i"
	if exact {
		suffix = "s"
	}
	return fmt.Sprintf(`"%s"%s`, strings.Replace(strings.Replace(value, `\`, `\\`, -1), `"`, `\"`, -1), suffix)
}

func escapeForTextSelector(text interface{}, exact bool) string {
	switch text := text.(type) {
	case *regexp.Regexp:
		pattern, flag := convertRegexp(text)
		return fmt.Sprintf(`/%s/%s`, pattern, flag)
	default:
		if exact {
			return fmt.Sprintf(`%ss`, escapeText(text.(string)))
		}
		return fmt.Sprintf(`%si`, escapeText(text.(string)))
	}
}

func escapeText(s string) string {
	builder := &strings.Builder{}
	encoder := json.NewEncoder(builder)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(s)
	return strings.TrimSpace(builder.String())
}

func getByAltTextSelector(text interface{}, exact bool) string {
	return getByAttributeTextSelector("alt", text, exact)
}

func getByAttributeTextSelector(attrName string, text interface{}, exact bool) string {
	switch text := text.(type) {
	case *regexp.Regexp:
		pattern, flag := convertRegexp(text)
		return fmt.Sprintf(`internal:attr=[%s=/%s/%s]`, attrName, pattern, flag)
	default:
		return fmt.Sprintf(`internal:attr=[%s=%s]`, attrName, escapeForAttributeSelector(text.(string), exact))
	}
}

func getByLabelSelector(text interface{}, exact bool) string {
	return fmt.Sprintf(`internal:label=%s`, escapeForTextSelector(text, exact))
}

func getByPlaceholderSelector(text interface{}, exact bool) string {
	return getByAttributeTextSelector("placeholder", text, exact)
}

func getByRoleSelector(role AriaRole, options ...LocatorGetByRoleOptions) string {
	props := make(map[string]string)
	if len(options) == 1 {
		if options[0].Checked != nil {
			props["checked"] = fmt.Sprintf("%v", options[0].Checked)
		}
		if options[0].Disabled != nil {
			props["disabled"] = fmt.Sprintf("%v", options[0].Disabled)
		}
		if options[0].Selected != nil {
			props["selected"] = fmt.Sprintf("%v", options[0].Selected)
		}
		if options[0].Expanded != nil {
			props["expanded"] = fmt.Sprintf("%v", options[0].Expanded)
		}
		if options[0].IncludeHidden != nil {
			props["includeHidden"] = fmt.Sprintf("%v", options[0].IncludeHidden)
		}
		if options[0].Level != nil {
			props["level"] = fmt.Sprintf("%d", options[0].Level)
		}
		if options[0].Name != nil {
			exact := false
			if options[0].Exact != nil {
				exact = *options[0].Exact
			}
			switch options[0].Name.(type) {
			case string:
				props["name"] = escapeForAttributeSelector(options[0].Name.(string), exact)
			case *regexp.Regexp:
				pattern, flag := convertRegexp(options[0].Name.(*regexp.Regexp))
				props["name"] = fmt.Sprintf(`/%s/%s`, pattern, flag)
			}
		}
		if options[0].Pressed != nil {
			props["pressed"] = fmt.Sprintf("%v", options[0].Pressed)
		}
	}
	propsStr := ""
	for k, v := range props {
		propsStr += "[" + k + "=" + v + "]"
	}
	return fmt.Sprintf("internal:role=%s%s", role, propsStr)
}

func getByTextSelector(text interface{}, exact bool) string {
	return fmt.Sprintf(`internal:text=%s`, escapeForTextSelector(text, exact))
}

func getByTestIdSelector(testIdAttributeName string, testId interface{}) string {
	switch testId := testId.(type) {
	case *regexp.Regexp:
		pattern, flag := convertRegexp(testId)
		return fmt.Sprintf(`internal:testid=[%s=/%s/%s]`, testIdAttributeName, pattern, flag)
	default:
		return fmt.Sprintf(`internal:testid=[%s=%s]`, testIdAttributeName, escapeForAttributeSelector(testId.(string), true))
	}
}

func getByTitleSelector(text interface{}, exact bool) string {
	return getByAttributeTextSelector("title", text, exact)
}

func getTestIdAttributeName() string {
	return testIdAttributeName
}

func setTestIdAttributeName(name string) {
	testIdAttributeName = name
}
