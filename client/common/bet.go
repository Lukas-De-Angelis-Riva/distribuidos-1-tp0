package common

type Bet struct {
	Name          string
	Surname       string
	Document      string
	BirthDate     string
	Number        string
}

func fromRecord(record []string) Bet {
	return Bet {
		Name: record[0],
		Surname: record[1],
		Document: record[2],
		BirthDate: record[3],
		Number: record[4],
	}
}