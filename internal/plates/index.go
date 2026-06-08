package plates

import (
	"sort"
	"strings"
)

type RackIndex struct {
	Root       string
	Plates     []Plate
	ByName     map[string][]Plate
	ByCategory map[string][]Plate
	TagCounts  map[string]int
}

func NewRackIndex(root string, plates []Plate) *RackIndex {
	index := &RackIndex{
		Root:       root,
		Plates:     append([]Plate(nil), plates...),
		ByName:     map[string][]Plate{},
		ByCategory: map[string][]Plate{},
		TagCounts:  map[string]int{},
	}
	sortPlates(index.Plates)
	for _, plate := range index.Plates {
		index.ByName[plate.Name] = append(index.ByName[plate.Name], plate)
		index.ByName[plate.Key()] = append(index.ByName[plate.Key()], plate)
		index.ByCategory[plate.Category] = append(index.ByCategory[plate.Category], plate)
		for _, tag := range plate.Tags {
			index.TagCounts[tag]++
		}
	}
	for category := range index.ByCategory {
		sortPlates(index.ByCategory[category])
	}
	for name := range index.ByName {
		sortPlates(index.ByName[name])
	}
	return index
}

func (r *RackIndex) Search(query string) []Plate {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return []Plate{}
	}
	var results []Plate
	for _, plate := range r.Plates {
		if plateMatches(plate, query) {
			results = append(results, plate)
		}
	}
	sortPlates(results)
	return results
}

func (r *RackIndex) Categories() map[string]int {
	counts := map[string]int{}
	for category, items := range r.ByCategory {
		counts[category] = len(items)
	}
	return counts
}

func (r *RackIndex) Tags() map[string]int {
	counts := map[string]int{}
	for tag, count := range r.TagCounts {
		counts[tag] = count
	}
	return counts
}

func (r *RackIndex) InCategory(category string) []Plate {
	plates := append([]Plate(nil), r.ByCategory[category]...)
	sortPlates(plates)
	return plates
}

func SortedMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortPlates(plates []Plate) {
	sort.SliceStable(plates, func(i, j int) bool {
		return plates[i].Key() < plates[j].Key()
	})
}

func plateMatches(plate Plate, query string) bool {
	fields := []string{plate.Name, plate.Category, plate.Description}
	fields = append(fields, plate.Tags...)
	for name, ingredient := range plate.Ingredients {
		fields = append(fields, name, ingredient.Description)
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}
