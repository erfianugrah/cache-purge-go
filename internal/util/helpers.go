package util

import (
	"strings"
)

// SplitCommaList splits a comma-separated string into a slice
func SplitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// MapFromZones creates a map of zones indexed by both name and ID
func MapFromZones(zones []interface{}) map[string]interface{} {
	zoneMap := make(map[string]interface{})

	for _, z := range zones {
		zone := z.(map[string]interface{})
		zoneMap[zone["name"].(string)] = zone
		zoneMap[zone["id"].(string)] = zone
	}

	return zoneMap
}

// ContainsString checks if a slice contains a specific string
func ContainsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

// FilterString removes a string from a slice
func FilterString(slice []string, str string) []string {
	result := make([]string, 0)
	for _, item := range slice {
		if item != str {
			result = append(result, item)
		}
	}
	return result
}

// StringSliceToSet converts a string slice to a map for O(1) lookups
func StringSliceToSet(slice []string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range slice {
		set[item] = true
	}
	return set
}

// FilterDuplicates removes duplicate strings from a slice
func FilterDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
