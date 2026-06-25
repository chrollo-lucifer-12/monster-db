package resp

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

var respNil = []byte("*-1\r\n")

var ErrIncomplete = errors.New("incomplete data")

func readSimpleString(data []byte) (string, int, error) {
	pos := 1
	for pos < len(data) && data[pos] != '\r' {
		pos++
	}

	if pos+1 >= len(data) || data[pos+1] != '\n' {
		return "", 0, ErrIncomplete
	}
	return string(data[1:pos]), pos + 2, nil
}

func readError(data []byte) (string, int, error) {
	return readSimpleString(data)
}

func readInt(data []byte) (int64, int, error) {
	pos := 1
	var value int64 = 0

	for pos < len(data) && data[pos] != '\r' {
		value = value*10 + int64(data[pos]-'0')
		pos++
	}

	if pos+1 >= len(data) || data[pos+1] != '\n' {
		return 0, 0, ErrIncomplete
	}
	return value, pos + 2, nil
}

func readLength(data []byte) (int, int, error) {
	pos := 0
	length := 0

	for pos < len(data) {
		b := data[pos]
		if b == '\r' {
			if pos+1 >= len(data) || data[pos+1] != '\n' {
				return 0, 0, ErrIncomplete
			}
			return length, pos + 2, nil
		}
		length = length*10 + int(b-'0')
		pos++
	}

	return 0, 0, ErrIncomplete
}

func readBulkString(data []byte) (string, int, error) {
	pos := 1
	length, delta, err := readLength(data[pos:])
	if err != nil {
		return "", 0, err
	}
	pos += delta

	if pos+length+2 > len(data) {
		return "", 0, ErrIncomplete
	}

	return string(data[pos:(pos + length)]), pos + length + 2, nil
}

func readArray(data []byte) (interface{}, int, error) {
	pos := 1

	count, delta, err := readLength(data[pos:])
	if err != nil {
		return nil, 0, err
	}
	pos += delta

	var elems []interface{} = make([]interface{}, count)

	for i := range elems {
		if pos >= len(data) {
			return nil, 0, ErrIncomplete
		}

		elem, delta, err := DecodeOne(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		elems[i] = elem
		pos += delta
	}

	return elems, pos, nil
}

func DecodeOne(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, ErrIncomplete
	}

	switch data[0] {
	case '+':
		return readSimpleString(data)
	case '-':
		return readError(data)
	case ':':
		return readInt(data)
	case '$':
		return readBulkString(data)
	case '*':
		return readArray(data)
	}

	return nil, 0, fmt.Errorf("invalid RESP type: %q (hex: %x)", data[0], data[0])
}

func Decode(data []byte) ([]interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, nil
	}

	var values []interface{} = make([]interface{}, 0)
	var index int = 0

	for index < len(data) {
		value, delta, err := DecodeOne(data[index:])
		if err != nil {

			if err == ErrIncomplete {
				return values, index, nil
			}
			return values, index, err
		}
		index = index + delta
		values = append(values, value)
	}

	return values, index, nil
}

func EncodeString(buf []byte, v string) []byte {
	var tmp [32]byte
	n := strconv.AppendInt(tmp[:0], int64(len(v)), 10)
	buf = append(buf, '$')
	buf = append(buf, n...)
	buf = append(buf, '\r', '\n')
	buf = append(buf, v...)
	return append(buf, '\r', '\n')
}

func EncodeInt(buf []byte, v int64) []byte {
	var tmp [32]byte
	n := strconv.AppendInt(tmp[:0], v, 10)
	buf = append(buf, ':')
	buf = append(buf, n...)
	return append(buf, '\r', '\n')
}

func EncodeArrayLen(buf []byte, l int) []byte {
	var tmp [32]byte
	n := strconv.AppendInt(tmp[:0], int64(l), 10)
	buf = append(buf, '*')
	buf = append(buf, n...)
	return append(buf, '\r', '\n')
}

func Encode(buf []byte, value any, isSimple bool) []byte {
	switch v := value.(type) {
	case nil:
		return append(buf, "*-1\r\n"...)
	case []byte:
		return append(buf, v...)
	case string:
		if isSimple {
			buf = append(buf, '+')
			buf = append(buf, v...)
			return append(buf, '\r', '\n')
		}
		return EncodeString(buf, v)

	case int:
		return EncodeInt(buf, int64(v))
	case int64:
		return EncodeInt(buf, v)
	case int32:
		return EncodeInt(buf, int64(v))
	case int16:
		return EncodeInt(buf, int64(v))
	case int8:
		return EncodeInt(buf, int64(v))

	case error:
		buf = append(buf, '-')
		buf = append(buf, v.Error()...)
		return append(buf, '\r', '\n')

	case []string:
		buf = EncodeArrayLen(buf, len(v))
		for _, item := range v {
			buf = EncodeString(buf, item)
		}
		return buf

	case []any:
		buf = EncodeArrayLen(buf, len(v))
		for _, item := range v {
			buf = Encode(buf, item, false)
		}
		return buf

	case [][]string:
		buf = EncodeArrayLen(buf, len(v))
		for _, row := range v {
			buf = EncodeArrayLen(buf, len(row))
			for _, item := range row {
				buf = EncodeString(buf, item)
			}
		}
		return buf
	}

	s := fmt.Sprint(value)
	return EncodeString(buf, s)
}

func EncodeExecArray(results [][]byte) []byte {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("*%d\r\n", len(results)))

	for _, r := range results {
		buf.Write(r)
	}

	return buf.Bytes()
}
