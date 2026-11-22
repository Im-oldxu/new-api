package model

import (
	"encoding/json"
	"slices"

	"github.com/Im-oldxu/new-api/common"
)

// Navigation options share rows with discontinued modules on older installations.
var navigationFields = map[string][]string{
	"header":   {"home", "console", "docs", "about"},
	"console":  {"enabled", "detail"},
	"personal": {"enabled", "personal", "security"},
	"admin":    {"enabled", "audit", "user", "setting"},
}

func navigationObject(raw string) (map[string]json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if raw != "" {
		if err := common.UnmarshalJsonStr(raw, &fields); err != nil || fields == nil {
			return nil, common.NewMessage("Invalid parameters")
		}
	}
	return fields, nil
}

// For reads, project current controls. For explicit writes, merge only current
// controls into the stored document, retaining unrecognized fields and sections.
func navigationOption(key, original string, patch *string) (string, error) {
	stored, err := navigationObject(original)
	if err != nil {
		return "", err
	}
	input := stored
	result := map[string]json.RawMessage{}
	if patch != nil {
		input, err = navigationObject(*patch)
		if err != nil {
			return "", err
		}
		result = stored
	}
	if key == "HeaderNavModules" {
		for name, v := range input {
			if !slices.Contains(navigationFields["header"], name) {
				if patch != nil {
					return "", ErrRetiredGatewayOption
				}
				continue
			}
			if patch != nil && string(v) != "true" && string(v) != "false" {
				return "", common.NewMessage("Invalid parameters")
			}
			result[name] = v
		}
	} else {
		for section, raw := range input {
			allowed, ok := navigationFields[section]
			if !ok || section == "header" {
				if patch != nil {
					return "", ErrRetiredGatewayOption
				}
				continue
			}
			fields, err := navigationObject(string(raw))
			if err != nil {
				return "", err
			}
			out := map[string]json.RawMessage{}
			if patch != nil {
				out, err = navigationObject(string(stored[section]))
				if err != nil {
					return "", err
				}
			}
			for name, v := range fields {
				if !slices.Contains(allowed, name) {
					if patch != nil {
						return "", ErrRetiredGatewayOption
					}
					continue
				}
				if patch != nil && string(v) != "true" && string(v) != "false" {
					return "", common.NewMessage("Invalid parameters")
				}
				out[name] = v
			}
			result[section], err = common.Marshal(out)
			if err != nil {
				return "", err
			}
		}
		if patch == nil && stored["console"] != nil {
			old, err := navigationObject(string(stored["console"]))
			if err != nil {
				return "", err
			}
			admin, err := navigationObject(string(result["admin"]))
			if err != nil {
				return "", err
			}
			if admin["audit"] == nil && (old["audit"] != nil || string(old["enabled"]) == "false") {
				admin["audit"] = old["audit"]
				if string(old["enabled"]) == "false" {
					admin["audit"] = json.RawMessage("false")
				}
				result["admin"], err = common.Marshal(admin)
				if err != nil {
					return "", err
				}
			}
		}
	}
	data, err := common.Marshal(result)
	return string(data), err
}
