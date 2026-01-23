package candidate

import "strings"

// FilterCandidates 根据过滤模式筛选候选词
func FilterCandidates(candidates []Candidate, mode string) []Candidate {
	if len(candidates) == 0 {
		return candidates
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "general":
		return filterCommonOnly(candidates)
	case "gb18030":
		return candidates
	case "smart":
		filtered := filterCommonOnly(candidates)
		if len(filtered) == 0 {
			return candidates
		}
		return filtered
	default:
		return candidates
	}
}

func filterCommonOnly(candidates []Candidate) []Candidate {
	result := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.IsCommon {
			result = append(result, c)
		}
	}
	return result
}
