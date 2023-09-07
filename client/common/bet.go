package common

type Bet struct {
	Name          string
	Surname       string
	Document      string
	BirthDate     string
	Number        string
}

// Crea una apuesta a partir de un record
// el record debe ser una lista de 5 elementos
// ordenados segun:
// * Nombre
// * Apellido
// * Documento
// * Fecha de nacimiento
// * Numero
// Tal cual aparecen en los archivos de datos.
func fromRecord(record []string) Bet {
	return Bet {
		Name: record[0],
		Surname: record[1],
		Document: record[2],
		BirthDate: record[3],
		Number: record[4],
	}
}