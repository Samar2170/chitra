package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

var validDurations = map[string]int{
	"week":    7,
	"day":     1,
	"month":   30,
	"quarter": 90,
}

var validMonths = map[string]int{
	"jan": 1,
	"feb": 2,
	"mar": 3,
	"apr": 4,
	"may": 5,
	"jun": 6,
	"jul": 7,
	"aug": 8,
	"sep": 9,
	"oct": 10,
	"nov": 11,
	"dec": 12,
}

func ParseDeadline(deadlineStr string) (time.Time, error) {
	// is deadline is
	now := time.Now().Local()

	if _, ok := validDurations[deadlineStr]; ok {
		deadline := now.Add(time.Hour * 24 * time.Duration(validDurations[deadlineStr]))
		deadline = time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.Local)
		return deadline, nil
	}
	var dayStr, monthStr string
	if len(deadlineStr) == 4 {
		dayStr = string(deadlineStr[0])
		if !(dayStr[0] >= '1' && dayStr[0] <= '9') {
			return time.Time{}, errors.New("invalid day")
		}
		monthStr = string(deadlineStr[1:4])
	} else if len(deadlineStr) == 5 {
		dayStr = deadlineStr[0:2]
		if !(dayStr[0] >= '1' && dayStr[0] <= '3') {
			return time.Time{}, errors.New("invalid day")
		}
		if !(dayStr[1] >= '0' && dayStr[1] <= '9') {
			return time.Time{}, errors.New("invalid day")
		}
		monthStr = string(deadlineStr[2:5])
	}
	month, ok := validMonths[monthStr]
	if !ok {
		return time.Time{}, errors.New("invalid month")
	}
	day, err := strconv.Atoi(dayStr)
	if err != nil {
		return time.Time{}, errors.New("invalid day")
	}

	// check if month has 31 days
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		if day > 31 {
			return time.Time{}, errors.New("invalid day")
		}
	case 4, 6, 9, 11:
		if day > 30 {
			return time.Time{}, errors.New("invalid day")
		}
	default:
		if day > 28 {
			return time.Time{}, errors.New("invalid day")
		}
	}

	deadline := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, time.Local)
	if deadline.Before(now) {
		deadline = deadline.AddDate(1, 0, 0)
	}
	return deadline, nil

}

func DateDiff(a, b time.Time) (year, month, day int) {
	// Ensure a is always before b
	if a.After(b) {
		a, b = b, a
	}

	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()

	year = y2 - y1
	month = int(m2 - m1)
	day = d2 - d1

	// Normalize negative days (borrow from previous month)
	if day < 0 {
		// days in month before 'b'
		// Set day to 0 (first of month), subtract 1 day to get last of prev month
		t := time.Date(y2, m2, 0, 0, 0, 0, 0, time.UTC)
		day += t.Day()
		month--
	}

	// Normalize negative months (borrow from year)
	if month < 0 {
		month += 12
		year--
	}

	return
}

// Optional: Helper to format it nicely (handling singular/plural)
func FormatDiff(years, months, days int) string {
	s := ""
	if years > 0 {
		s += fmt.Sprintf("%d year%s ", years, plural(years))
	}
	if months > 0 {
		s += fmt.Sprintf("%d month%s ", months, plural(months))
	}
	s += fmt.Sprintf("%d day%s", days, plural(days))
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
