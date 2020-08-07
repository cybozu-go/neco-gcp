package necogcpfunctions

import "time"

var jpHolidays = []string{
	"20200810",
	"20200921",
	"20200922",
	"20201103",
	"20201123",
	"20201229",
	"20201230",
	"20201231",
	"20210101",
	"20210102",
	"20210103",
}

func getDateStrInJST() (string, error) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return "", err
	}
	return time.Now().In(loc).Format("20060102"), nil
}

func isHoliday(target string, holidays []string) bool {
	for _, h := range holidays {
		if target == h {
			return true
		}
	}
	return false
}
