package utils

import "strconv"

const (
	MORE_THEN = iota
	MORE_EQUAL_THEN
	LESS_THEN
	LESS_EQUAL_THEN
	EQUAL
	ALL
)

func ParseVersion(ver string) (int, string) {
	if len(ver) >= 2 {
		if ver[:2] == "<=" {
			return LESS_EQUAL_THEN, ver[2:]
		} else if ver[:2] == ">=" {
			return MORE_EQUAL_THEN, ver[2:]
		} else if ver[:1] == "<" {
			return LESS_THEN, ver[1:]
		} else if ver[:1] == ">" {
			return MORE_THEN, ver[1:]
		} else if ver[:1] == "=" {
			return EQUAL, ver[1:]
		}
	}

	return ALL, ver
}

func CompareVersions(have, need string, op int) (bool, error) {
	have_f, err := strconv.ParseFloat(have, 64)
	if err != nil {
		return false, err
	}
	if need == "" {
		return true, nil
	}
	need_f, err := strconv.ParseFloat(need, 64)
	if err != nil {
		return false, err
	}
	switch op {
	case MORE_THEN:
		if have_f > need_f {
			return true, nil
		} else {
			return false, nil
		}
	case MORE_EQUAL_THEN:
		if have_f >= need_f {
			return true, nil
		} else {
			return false, nil
		}
	case LESS_THEN:
		if have_f < need_f {
			return true, nil
		} else {
			return false, nil
		}
	case LESS_EQUAL_THEN:
		if have_f <= need_f {
			return true, nil
		} else {
			return false, nil
		}
	case EQUAL:
		if have_f == need_f {
			return true, nil
		} else {
			return false, nil
		}
	case ALL:
		return true, nil
	}
	return false, nil
}
