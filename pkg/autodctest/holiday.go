package autodctest

import (
	"slices"
	"time"
)

func getDateStrInJST() (string, error) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return "", err
	}
	return time.Now().In(loc).Format("20060102"), nil
}

func isHoliday(target string, holidays []string) bool {
	return slices.Contains(holidays, target)
}
