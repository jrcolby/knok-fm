package domain

import "strings"

// GetPlatformConstraintSQL generates the SQL constraint for platform validation
func GetPlatformConstraintSQL() string {
	platforms := GetValidPlatforms()
	quotedPlatforms := make([]string, len(platforms))

	for i, platform := range platforms {
		quotedPlatforms[i] = "'" + platform + "'"
	}

	return "CHECK (platform IN (" + strings.Join(quotedPlatforms, ", ") + "))"
}
