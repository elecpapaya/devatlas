package saramin

import (
	"fmt"
	"time"
)

func ExampleJobSearchParams_Encode() {
	params := JobSearchParams{
		JobCd:      []string{"84", "92"},
		Sr:         []string{"directhire"},
		Fields:     []string{"posting-date", "expiration-date", "count"},
		UpdatedMin: time.Unix(1700000000, 0),
		UpdatedMax: time.Unix(1700086400, 0),
		Count:      10,
		Sort:       "ud",
	}

	values, err := params.Encode("test-key")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(values.Encode())
	// Output:
	// access-key=test-key&count=10&fields=posting-date%2Cexpiration-date%2Ccount&job_cd=84%2C92&sort=ud&sr=directhire&updated_max=1700086400&updated_min=1700000000
}
