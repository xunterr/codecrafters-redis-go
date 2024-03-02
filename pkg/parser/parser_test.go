package parser

import (
	"testing"
)

func TestParseArray(t *testing.T) {
	scanner := NewScanner("*2\r\n$4\r\nECHO\r\n$3\r\nABC\r\n")
	tokens := scanner.ScanTokens()
	parser := Parser{
		tokens:  tokens,
		current: 0,
	}

	data := parser.Parse()

	if data.DataType != ArrayData {
		t.Errorf("Wrong container data type. Have: %d, want: %d (ArrayData)", data.DataType, ArrayData)
	}

	array := data.Value.([]Data)
	if len(array) != 2 {
		t.Errorf("Wrong array length. Have: %d, want: 2", len(array))
	}

	for _, data := range array {
		if data.DataType != StringData {
			t.Errorf("Wrong array content type. Have: %d, want: %d (StringData)", data.DataType, StringData)
		}
	}
}

func TestToMap(t *testing.T) {
	data := &Data{
		DataType: ArrayData,
		Value: Data{
			DataType: StringData,
			Value:    "Hi",
		},
	}
	p, err := data.ToMap()
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(p)
}
