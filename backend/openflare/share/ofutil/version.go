// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package ofutil

import (
	"strconv"
	"strings"
)

const gitDescribeMinIdentifiers = 2

type versionInfo struct {
	valid               bool
	isDev               bool
	numbers             []int
	prerelease          []string
	gitDescribeDistance int
	gitDescribeTail     []string
}

func parseVersionInfo(version string) versionInfo {
	normalized := strings.TrimSpace(strings.TrimPrefix(version, "v"))
	if normalized == "" || normalized == "dev" {
		return versionInfo{isDev: strings.EqualFold(normalized, "dev")}
	}
	base := normalized
	prerelease := ""
	if separator := strings.IndexRune(normalized, '-'); separator >= 0 {
		base = normalized[:separator]
		prerelease = normalized[separator+1:]
	}

	segments := strings.Split(base, ".")
	parts := make([]int, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			parts = append(parts, 0)
			continue
		}

		numeric := strings.Builder{}
		for _, r := range segment {
			if r < '0' || r > '9' {
				break
			}
			numeric.WriteRune(r)
		}
		if numeric.Len() == 0 {
			parts = append(parts, 0)
			continue
		}
		value, err := strconv.Atoi(numeric.String())
		if err != nil {
			return versionInfo{}
		}
		parts = append(parts, value)
	}
	info := versionInfo{valid: len(parts) > 0, numbers: parts}
	if prerelease != "" {
		identifiers := splitPrereleaseIdentifiers(prerelease)
		if distance, tail, ok := parseGitDescribeIdentifiers(identifiers); ok {
			info.gitDescribeDistance = distance
			info.gitDescribeTail = tail
		} else {
			info.prerelease = identifiers
		}
	}
	return info
}

func parseGitDescribeIdentifiers(identifiers []string) (int, []string, bool) {
	if len(identifiers) < gitDescribeMinIdentifiers {
		return 0, nil, false
	}
	distance, err := strconv.Atoi(strings.TrimSpace(identifiers[0]))
	if err != nil || distance <= 0 {
		return 0, nil, false
	}
	commitToken := strings.TrimSpace(identifiers[1])
	if commitToken == "" || !strings.HasPrefix(strings.ToLower(commitToken), "g") {
		return 0, nil, false
	}
	return distance, identifiers[1:], true
}

func splitPrereleaseIdentifiers(value string) []string {
	parts := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == '.' || r == '-'
	})
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return filtered
}

func CompareVersions(local, remote string) int {
	left := parseVersionInfo(local)
	right := parseVersionInfo(remote)
	if left.isDev {
		if right.valid {
			return -1
		}
		return 0
	}
	if !left.valid || !right.valid {
		return 0
	}

	if result := compareVersionNumbers(left, right); result != 0 {
		return result
	}
	if result := compareGitDescribeDistance(left, right); result != 0 {
		return result
	}
	if left.gitDescribeDistance > 0 || right.gitDescribeDistance > 0 {
		return compareGitDescribeTails(left, right)
	}
	return comparePrereleaseIdentifiers(left, right)
}

func compareVersionNumbers(left, right versionInfo) int {
	maxLen := max(len(right.numbers), len(left.numbers))
	for index := range maxLen {
		leftValue := 0
		rightValue := 0
		if index < len(left.numbers) {
			leftValue = left.numbers[index]
		}
		if index < len(right.numbers) {
			rightValue = right.numbers[index]
		}
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}
	return 0
}

func compareGitDescribeDistance(left, right versionInfo) int {
	if left.gitDescribeDistance == right.gitDescribeDistance {
		return 0
	}
	if left.gitDescribeDistance < right.gitDescribeDistance {
		return -1
	}
	return 1
}

func compareGitDescribeTails(left, right versionInfo) int {
	maxLen := max(len(right.gitDescribeTail), len(left.gitDescribeTail))
	for index := range maxLen {
		if index >= len(left.gitDescribeTail) {
			return -1
		}
		if index >= len(right.gitDescribeTail) {
			return 1
		}
		if left.gitDescribeTail[index] < right.gitDescribeTail[index] {
			return -1
		}
		if left.gitDescribeTail[index] > right.gitDescribeTail[index] {
			return 1
		}
	}
	return 0
}

func comparePrereleaseIdentifiers(left, right versionInfo) int {
	if len(left.prerelease) == 0 && len(right.prerelease) == 0 {
		return 0
	}
	if len(left.prerelease) == 0 {
		return 1
	}
	if len(right.prerelease) == 0 {
		return -1
	}

	maxLen := max(len(right.prerelease), len(left.prerelease))
	for index := range maxLen {
		if index >= len(left.prerelease) {
			return -1
		}
		if index >= len(right.prerelease) {
			return 1
		}
		if result := comparePrereleasePart(left.prerelease[index], right.prerelease[index]); result != 0 {
			return result
		}
	}
	return 0
}

func comparePrereleasePart(leftPart, rightPart string) int {
	leftNumber, leftErr := strconv.Atoi(leftPart)
	rightNumber, rightErr := strconv.Atoi(rightPart)
	switch {
	case leftErr == nil && rightErr == nil:
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
	case leftErr == nil:
		return -1
	case rightErr == nil:
		return 1
	default:
		if leftPart < rightPart {
			return -1
		}
		if leftPart > rightPart {
			return 1
		}
	}
	return 0
}
