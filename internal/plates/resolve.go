package plates

import "sort"

type Resolution struct {
	Values  map[string]string
	Missing []string
}

func Resolve(plate Plate, pantry, workspace, session map[string]string) Resolution {
	values := map[string]string{}
	for key, ingredient := range plate.Ingredients {
		if ingredient.Default != "" {
			values[key] = ingredient.Default
		}
	}
	merge(values, pantry)
	merge(values, workspace)
	merge(values, session)

	var missing []string
	for key, ingredient := range plate.Ingredients {
		if ingredient.Required && values[key] == "" {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return Resolution{Values: values, Missing: missing}
}

func merge(dst, src map[string]string) {
	for key, value := range src {
		dst[key] = value
	}
}
