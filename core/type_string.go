package core

func deduceTypeEncoding(v string) (uint8, uint8) {
	oType := OBJ_TYPE_STRING
	if isInt(v) {
		return oType, OBJ_ENCODING_INT
	}

	if len(v) <= 44 {
		return oType, OBJ_ENCODING_EMBSTR
	}

	return oType, OBJ_ENCODING_RAW
}

func isInt(v string) bool {
	if len(v) == 0 {
		return false
	}
	i := 0
	if v[0] == '-' {
		i = 1
		if len(v) == 1 {
			return false
		}
	}
	for ; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
	}
	return true
}
