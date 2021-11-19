package necogcp

// Instance settings for the "auto-dctest" function.
// The instances will be created with the specified zone and machine type.
const (
	autoDCTestMachineType = "n1-highmem-32"
	autoDCTestZone        = "asia-northeast1-c"
)

// The holiday list for the "auto-dctest" function.
// On the days listed here, the auto instance creation will be skipped.
// Please update annually :)
var autoDCTestJPHolidays = []string{
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
