//+build ignore

package juroku

// type Top2 struct {
// 	top1 byte
// 	top1Count byte
// 	top2 byte
// 	top2Count byte
// 	lastPos byte
// 	d [6]byte
// }

func Sort6(d []float64) {
	// bounds check elimination
	if len(d) < 6 {
		panic("must be at least 6!!!")
	}

	var a, b float64

	if d[1] < d[2] {
		a = d[1]
	} else {
		a = d[2]
	}
	if d[1] < d[2] {
		b = d[2]
	} else {
		b = d[1]
	}
	d[1] = a
	d[2] = b

	if d[4] < d[5] {
		a = d[4]
	} else {
		a = d[5]
	}
	if d[4] < d[5] {
		b = d[5]
	} else {
		b = d[4]
	}
	d[4] = a
	d[5] = b

	if d[0] < d[2] {
		a = d[0]
	} else {
		a = d[2]
	}
	if d[0] < d[2] {
		b = d[2]
	} else {
		b = d[0]
	}
	d[0] = a
	d[2] = b

	if d[3] < d[5] {
		a = d[3]
	} else {
		a = d[5]
	}
	if d[3] < d[5] {
		b = d[5]
	} else {
		b = d[3]
	}
	d[3] = a
	d[5] = b

	if d[0] < d[1] {
		a = d[0]
	} else {
		a = d[1]
	}
	if d[0] < d[1] {
		b = d[1]
	} else {
		b = d[0]
	}
	d[0] = a
	d[1] = b

	if d[3] < d[4] {
		a = d[3]
	} else {
		a = d[4]
	}
	if d[3] < d[4] {
		b = d[4]
	} else {
		b = d[3]
	}
	d[3] = a
	d[4] = b

	if d[1] < d[4] {
		a = d[1]
	} else {
		a = d[4]
	}
	if d[1] < d[4] {
		b = d[4]
	} else {
		b = d[1]
	}
	d[1] = a
	d[4] = b

	if d[0] < d[3] {
		a = d[0]
	} else {
		a = d[3]
	}
	if d[0] < d[3] {
		b = d[3]
	} else {
		b = d[0]
	}
	d[0] = a
	d[3] = b

	if d[2] < d[5] {
		a = d[2]
	} else {
		a = d[5]
	}
	if d[2] < d[5] {
		b = d[5]
	} else {
		b = d[2]
	}
	d[2] = a
	d[5] = b

	if d[1] < d[3] {
		a = d[1]
	} else {
		a = d[3]
	}
	if d[1] < d[3] {
		b = d[3]
	} else {
		b = d[1]
	}
	d[1] = a
	d[3] = b

	if d[2] < d[4] {
		a = d[2]
	} else {
		a = d[4]
	}
	if d[2] < d[4] {
		b = d[4]
	} else {
		b = d[2]
	}
	d[2] = a
	d[4] = b

	if d[2] < d[3] {
		a = d[2]
	} else {
		a = d[3]
	}
	if d[2] < d[3] {
		b = d[3]
	} else {
		b = d[2]
	}
	d[2] = a
	d[3] = b

}
