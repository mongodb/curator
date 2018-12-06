package poplar

func isMoreThanOneTrue(in []bool) bool {
	count := 0
	for _, v := range in {
		if v {
			count++
		}
		if count > 1 {
			return true
		}
	}

	return false
}
