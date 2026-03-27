package admin

// transformProductArgs converts agent-friendly MCP arguments into the format
// expected by the REST API (CreateProductRequest / UpdateProductRequest).
func transformProductArgs(args map[string]any) map[string]any {
	// price → price_net + price_gross
	if price, ok := args["price"]; ok {
		p := toInt(price)
		if _, has := args["price_net"]; !has {
			args["price_net"] = p
		}
		if _, has := args["price_gross"]; !has {
			args["price_gross"] = p
		}
		delete(args, "price")
	}

	// Default currency
	if _, ok := args["currency"]; !ok {
		args["currency"] = "EUR"
	}

	// Extract top-level name/slug/description before they are removed
	name, _ := args["name"].(string)
	slug, _ := args["slug"].(string)
	desc, _ := args["description"].(string)

	delete(args, "name")
	delete(args, "slug")
	delete(args, "description")

	// Transform translations map→array
	args = transformTranslations(args)

	// If no translations exist but name/slug were given, create a default en-US translation
	if _, ok := args["translations"]; !ok {
		if name != "" || slug != "" {
			t := map[string]any{"locale": "en-US"}
			if name != "" {
				t["name"] = name
			}
			if slug != "" {
				t["slug"] = slug
			}
			if desc != "" {
				t["description"] = desc
			}
			args["translations"] = []any{t}
		}
	}

	return args
}

// transformTranslations converts a locale-keyed translations object into the
// array format expected by the REST API:
//
//	{"de-DE": {"name":"..."}} → [{"locale":"de-DE","name":"..."}]
func transformTranslations(args map[string]any) map[string]any {
	raw, ok := args["translations"]
	if !ok {
		return args
	}

	// Already an array — pass through
	if _, isArr := raw.([]any); isArr {
		return args
	}

	obj, isMap := raw.(map[string]any)
	if !isMap {
		return args
	}

	arr := make([]any, 0, len(obj))
	for locale, v := range obj {
		entry, entryOk := v.(map[string]any)
		if !entryOk {
			continue
		}
		entry["locale"] = locale
		arr = append(arr, entry)
	}
	args["translations"] = arr
	return args
}

// transformVariantArgs converts agent-friendly variant arguments into the
// REST API format (price → price_net + price_gross).
func transformVariantArgs(args map[string]any) map[string]any {
	if price, ok := args["price"]; ok {
		p := toInt(price)
		if _, has := args["price_net"]; !has {
			args["price_net"] = p
		}
		if _, has := args["price_gross"]; !has {
			args["price_gross"] = p
		}
		delete(args, "price")
	}
	return args
}

// transformCategoryArgs converts agent-friendly category arguments into the
// REST API format (translations map→array, top-level name/slug/description removal).
func transformCategoryArgs(args map[string]any) map[string]any {
	name, _ := args["name"].(string)
	slug, _ := args["slug"].(string)
	desc, _ := args["description"].(string)

	delete(args, "name")
	delete(args, "slug")
	delete(args, "description")

	args = transformTranslations(args)

	if _, ok := args["translations"]; !ok {
		if name != "" || slug != "" {
			t := map[string]any{"locale": "en-US"}
			if name != "" {
				t["name"] = name
			}
			if slug != "" {
				t["slug"] = slug
			}
			if desc != "" {
				t["description"] = desc
			}
			args["translations"] = []any{t}
		}
	}

	return args
}

// transformPropertyGroupArgs converts agent-friendly property group arguments
// into the REST API format (name → translations array).
func transformPropertyGroupArgs(args map[string]any) map[string]any {
	name, _ := args["name"].(string)
	delete(args, "name")

	args = transformTranslations(args)

	if _, ok := args["translations"]; !ok {
		if name != "" {
			args["translations"] = []any{
				map[string]any{"locale": "en-US", "name": name},
			}
		}
	}

	return args
}

// transformPropertyOptionArgs converts agent-friendly property option arguments
// into the REST API format (name → translations array, keeps color_hex at top level).
func transformPropertyOptionArgs(args map[string]any) map[string]any {
	name, _ := args["name"].(string)
	delete(args, "name")

	args = transformTranslations(args)

	if _, ok := args["translations"]; !ok {
		if name != "" {
			args["translations"] = []any{
				map[string]any{"locale": "en-US", "name": name},
			}
		}
	}

	return args
}

// transformAttributeArgs converts agent-friendly attribute arguments
// into the REST API format (name/description → translations array).
func transformAttributeArgs(args map[string]any) map[string]any {
	name, _ := args["name"].(string)
	desc, _ := args["description"].(string)
	delete(args, "name")
	delete(args, "description")

	args = transformTranslations(args)

	if _, ok := args["translations"]; !ok {
		if name != "" {
			t := map[string]any{"locale": "en-US", "name": name}
			if desc != "" {
				t["description"] = desc
			}
			args["translations"] = []any{t}
		}
	}

	return args
}

// transformAttributeOptionArgs converts agent-friendly attribute option arguments
// into the REST API format (name → translations array).
func transformAttributeOptionArgs(args map[string]any) map[string]any {
	name, _ := args["name"].(string)
	delete(args, "name")

	args = transformTranslations(args)

	if _, ok := args["translations"]; !ok {
		if name != "" {
			args["translations"] = []any{
				map[string]any{"locale": "en-US", "name": name},
			}
		}
	}

	return args
}

// toInt converts a value to int, handling float64 (JSON default) and int.
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
