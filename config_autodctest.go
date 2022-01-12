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
	"20220110",
	"20220211",
	"20220223",
	"20220321",
	"20220429",
	"20220503",
	"20220504",
	"20220505",
	"20220718",
	"20220811",
	"20220919",
	"20220923",
	"20221010",
	"20221103",
	"20221123",
	"20221229",
	"20221230",
	"20221231",
	"20230101",
	"20230102",
	"20230103",
}
