package pqinterval

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseErr is returned on a failure to parse a
// postgres result into an Interval or Duration.
type ParseErr struct {
	String string
	Cause  error
}

func parse(s string) (Interval, error) {
	chunks := strings.Split(s, " ")

	ival := Interval{}
	var negTime bool

	// the space delimited sections of a postgres-formatted interval
	// come in pairs until the time portion: "3 years 2 days 04:15:47"
	if len(chunks)%2 == 1 {
		t := chunks[len(chunks)-1]
		chunks = chunks[:len(chunks)-1]

		switch t[0] {
		case '-':
			negTime = true
			t = t[1:]
		case '+':
			t = t[1:]
		}

		// hh:mm:ss[.uuuuuu]
		timeChunks := strings.Split(t, ":")
		if len(timeChunks) != 3 {
			return ival, ParseErr{s, nil}
		}

		hrs, err := strconv.Atoi(timeChunks[0])
		if err != nil {
			return ival, ParseErr{s, err}
		}
		if negTime {
			hrs = -hrs
		}

		mins, err := strconv.Atoi(timeChunks[1])
		if err != nil || mins > 59 || mins < 0 {
			return ival, ParseErr{s, err}
		}

		secParts := strings.SplitN(timeChunks[2], ".", 2)
		secs, err := strconv.Atoi(secParts[0])
		if err != nil || secs > 59 || secs < 0 {
			return ival, ParseErr{s, err}
		}

		var us int
		if len(secParts) > 1 {
			usPart := secParts[1]
			usPart += strings.Repeat("0", 6-len(usPart))
			us, err = strconv.Atoi(usPart)
			if err != nil {
				return ival, ParseErr{s, err}
			}
		}

		us += secs*usPerSec + mins*usPerMin

		ival.hrs = int32(hrs)
		ival.us = uint32(us)
	}

	for len(chunks) > 0 {
		t := chunks[0]
		unit := chunks[1]
		chunks = chunks[2:]

		n, err := strconv.Atoi(t)
		if err != nil {
			return Interval{}, ParseErr{s, err}
		}

		switch unit {
		case "year", "years":
			if n < 0 {
				n *= -1
				n |= yrSignBit
			}
			ival.yrs = uint32(n)

		case "mon", "mons":
			ival.hrs += int32(24 * daysPerMon * n)

		case "day", "days":
			ival.hrs += int32(24 * n)

		default:
			return Interval{}, ParseErr{s, nil}
		}
	}

	if negTime {
		ival.yrs |= usSignBit
	}

	return ival, nil
}

// Error implements the error interface.
func (pe ParseErr) Error() string {
	return fmt.Sprintf("pqinterval: Error parsing %q: %s", pe.String, pe.Cause)
}
