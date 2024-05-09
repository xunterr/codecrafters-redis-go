package parser

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/utils"
	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	tests := []utils.Test[string, Data]{
		{Name: "Parse array", Input: "*2\r\n$4\r\nECHO\r\n$3\r\nABC\r\n", Want: Data{
			dataType: Array,
			array: []Data{
				{
					dataType: BulkString,
					string:   "ECHO",
				},
				{
					dataType: BulkString,
					string:   "ABC",
				},
			},
		}},
		{Name: "Parse integer", Input: ":456\r\n", Want: Data{
			dataType: Integer,
			integer:  456,
		}},
	}

	for _, e := range tests {
		parser := Parser{
			source:  e.Input,
			current: 0,
		}

		data, err := parser.Parse()

		if err != nil {
			t.Error(err.Error())
		}

		if data.dataType != e.Want.dataType {
			t.Errorf("Wrong container data type. Have: %d, want: %d", data.dataType, e.Want.dataType)
		}

		if !cmp.Equal(*data, e.Want, cmp.AllowUnexported(Data{})) {
			t.Errorf(e.ToString(*data))
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
