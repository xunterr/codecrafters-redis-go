package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseArray(t *testing.T) {
	source := "*2\r\n$4\r\nECHO\r\n$3\r\nABC\r\n"
	parser := Parser{
		source:  source,
		current: 0,
	}

	data, err := parser.Parse()

	if err != nil {
		t.Error(err.Error())
	}

	if data.dataType != Array {
		t.Errorf("Wrong container data type. Have: %d, want: %d (Array)", data.dataType, Array)
	}

	array := data.array
	if len(array) != 2 {
		t.Errorf("Wrong array length. Have: %d, want: 2", len(array))
	}

	for _, data := range array {
		if data.dataType != BulkString {
			t.Errorf("Wrong array content type. Have: %s, want: %s (BulkString)", string(data.dataType), string(BulkString))
		}
	}
}

func TestToMap(t *testing.T) {
	data := &Data{
		dataType: Array,
		array: []Data{
			{
				dataType: String,
				string:   "Hi",
			},
		},
	}

	p, err := data.ToMap()
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(p)
}

func TestFlat(t *testing.T) {
	data := &Data{
		dataType: Array,
		array: []Data{
			{
				dataType: String,
				string:   "Hi",
			},
		},
	}
	res := data.Flat()
	expected := []string{"Hi"}

	if !cmp.Equal(res, expected) {
		t.Errorf("Wrong flat result. Have: %v, want: %v", res, expected)
	}
}

func TestMarshalSimple(t *testing.T) {
	data := Data{
		dataType: String,
		string:   "Hello",
	}
	want := "+Hello\r\n"
	res := data.marshalSimple()
	if string(res) != want {
		t.Errorf("Wrong marshal result. Have: %s, want: %s", string(res), want)
	}
}

func TestMarshalBulk(t *testing.T) {
	data := Data{
		dataType: BulkString,
		string:   "Hello",
	}
	want := "$5\r\nHello\r\n"
	res := data.marshalBulk()
	if string(res) != want {
		t.Errorf("Wrong marshal result. Have: %s, want: %s", string(res), want)
	}
}
