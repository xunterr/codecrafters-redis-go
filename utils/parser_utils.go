package utils

import (
	"regexp"
)

func IsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func IsAlfa(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_' || c == ' '
}

func IsAlfaNumeric(c byte) bool {
	return IsDigit(c) || IsAlfa(c)
}

func IsSpecial(c byte) bool {
	isSpecial := regexp.MustCompile("[^A-Za-z0-9]").MatchString(string(c))
	return isSpecial && c != '\r' && c != '\n'
}
