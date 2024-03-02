package utils

func IsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func IsAlfa(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		c == '_'
}
