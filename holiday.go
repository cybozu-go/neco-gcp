package necogcp

import "time"

var jpHolidays = []string{
	"20210111",
	"20210211",
	"20210223",
	"20210320",
	"20210429",
	"20210503",
	"20210504",
	"20210505",
	"20210722",
	"20210723",
	"20210808",
	"20210809",
	"20210920",
	"20210923",
	"20211103",
	"20211123",
	"20211229",
	"20211230",
	"20211231",
	"20220101",
	"20220102",
	"20220103",
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
