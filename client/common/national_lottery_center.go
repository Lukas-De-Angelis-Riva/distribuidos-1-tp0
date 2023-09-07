package common

import(
    "net"
    "encoding/binary"
    "errors"
    "strconv"

    log "github.com/sirupsen/logrus"
)

const BATCH_TYPE = 'Z'
const BET_TYPE = 'B'
const AGENCY_NAME_TYPE = 'A'
const NAME_TYPE = 'N'
const LAST_NAME_TYPE = 'L'
const DOCUMENT_TYPE = 'D'
const BIRTHDATE_TYPE = 'H'
const NUMBER_TYPE = 'U'

const POLL_TYPE = 'P'
const FINISH_TYPE = 'F'

const WINNERS_TYPE = 'W'
const AWAIT_TYPE = 'Y'
const OK_TYPE = 'O'

const ERROR = -1
const WAIT = 0
const INFO = 1

// Entidad que maneja la comunicacion con el centro de loteria nacional
type NationalLotteryCenter struct {
    conn net.Conn 
    ID string
}

// Crea el comunicador con la central. genera un socket tcp/ip
//  con el que se comunicara con el servidor
func NewNationalLotteryCenter(ID string, ServerAddress string) *NationalLotteryCenter {
    conn, err := net.Dial("tcp", ServerAddress)
    if err != nil {
        log.Fatalf(
            "action: connect | result: fail | client_id: %v | error: %v",
            ID,
            err,
        )
    }

    center := &NationalLotteryCenter{
        conn: conn,
        ID: ID,
    }

    return center
}

// Envia todos los bytes en data por la conexióon conn.
// Previene las anomalias de short-write
//
// Si no hay error se devuelve nil, en caso de error
//  este es devuelto
func sendData(conn net.Conn, data []byte) error {
    for totalSent := 0; totalSent < len(data); {
        sent, err := conn.Write(data[totalSent:])
        if err != nil {
            return err
        }
        totalSent += sent
    }

    return nil
}

// Serializa un campo string a bytes, y por delante pone
//  el indicador del tipo siguiente el protocolo TLV propuesto
//
// Devuelve los bytes serializados
func serializeString(field_type byte, field string) []byte {
    serialized := []byte{}
    serialized = append(serialized, field_type)

    length := make([]byte, 4)
    binary.BigEndian.PutUint32(length, uint32(len(field)))
    serialized = append(serialized, length...)

    return append(serialized, []byte(field)...)
}

// Serializa por completo una apuesta, utilizando el protocolo TLV propuesto
// El orden de los campos no interesa
func (p *NationalLotteryCenter) serializeBet(bet Bet) []byte {
    serialized := []byte{}
    serialized = append(serialized, serializeString(AGENCY_NAME_TYPE, p.ID)...)
    serialized = append(serialized, serializeString(NAME_TYPE, bet.Name)...)
    serialized = append(serialized, serializeString(LAST_NAME_TYPE, bet.Surname)...)
    serialized = append(serialized, serializeString(DOCUMENT_TYPE, bet.Document)...)
    serialized = append(serialized, serializeString(BIRTHDATE_TYPE, bet.BirthDate)...)
    serialized = append(serialized, serializeString(NUMBER_TYPE, bet.Number)...)

    first_part := []byte{}
    first_part = append(first_part, BET_TYPE)
    length := make([]byte, 4)
    binary.BigEndian.PutUint32(length, uint32(len(serialized)))
    first_part = append(first_part, length...)

    data := append(first_part, serialized...)
    return data
}

// Envia una apuesta a traves del socket de comunicacion
// 
// La apuesta es serializada segun el protocolo TLV propuesto y enviada
//  por completo a traves del socket-
//
// Si no hay error se devuelve nil, en caso de error se devuelve este
func (p *NationalLotteryCenter) sendBet(bet Bet) error {
    data := p.serializeBet(bet)
    err := sendData(p.conn, data)
    if err != nil {
        log.Errorf("action: send_message | result: fail | client_id: %v | error: %v", p.ID, err)
        return err
    }

    return nil
}

// Envia un conjunto de apuestas conjuntamente a traves
//  del socket de comunicacion
// El envio se hace en conjunto, es decir los datos se envian
//  uno tras otro en un tira de bits simultaneamente
func (p *NationalLotteryCenter) sendBatch(batch []Bet) error {
    first_part := []byte{}
    // Se envia el indicador
    first_part = append(first_part, BATCH_TYPE)
    length := make([]byte, 4)
    // Se envia cuantas apuestas seran enviadas
    binary.BigEndian.PutUint32(length, uint32(len(batch)))
    first_part = append(first_part, length...)

    data := []byte{}
    for _, bet := range batch {
        serial_bet := p.serializeBet(bet)
        data = append(data, serial_bet...)
    }

    data = append(first_part, data...)

    return sendData(p.conn, data)
}

// Lee una cantidad especifica de bytes de un socket
// Evita anomalias de short-read
//
// En caso de error se devuelve
func readAll(conn net.Conn, bytesToRead int) ([]byte, error) {

    bytesReaded := 0
    data := make([]byte, bytesToRead)

    for bytesReaded < bytesToRead{
        n, err := conn.Read(data[bytesReaded:])
        if err != nil {
            return data, err
        }
        bytesReaded += n
    }
    return data, nil
}

// Lee del socket un byte a la espera de la confirmacion
// 
// Si ocurre un error en la lectura o  no se leyo lo esperado,
//  se devuelve un error.
func (p *NationalLotteryCenter) waitConfirmation() error {

    confirmation, err := readAll(p.conn, 1) // leer el tipo
    if err != nil {
        return errors.New("Confirmation error: cannot read TYPE") // probar si funciona %v, err
    }

    if confirmation[0] == byte(OK_TYPE){
        return nil
    } else {
        return errors.New("Confirmation error: NOT OK")
    }
}

// Envia por el socket el byte correspondiente a cortar la comunicacion
//  segun lo establecido en el protocolo TLV propuesto
func (p *NationalLotteryCenter) Finish() error {
    goodbye := []byte{}
    goodbye = append(goodbye, FINISH_TYPE)

    err := sendData(p.conn, goodbye)
    return err

}

// Cierra la conexion con el servidor
func (p *NationalLotteryCenter) Close() {
    p.conn.Close()
}

// Lee del socket un documento. Dicho documento sera devuelto como
//  el primer elemento si no ocurre un error.
// 
// Si ocurre algun error sera devuelto como el segundo elemento
func (p *NationalLotteryCenter) ReadDocument() (string, error) {
    tlv_type, err := readAll(p.conn, 1) // leo el tipo
    if err != nil {
        return "", err
    }

    if tlv_type[0] == byte(DOCUMENT_TYPE) {
        document_len_d, err := readAll(p.conn, 4) // leer el tamaño
        if err != nil {
            return "", err
        }
        document_len := int(binary.BigEndian.Uint32(document_len_d))

        document_d, err := readAll(p.conn, document_len)
        if err != nil {
            return "", err
        }

        document := string(document_d)
        return document, nil

    } else {
        return "", errors.New("Protocol error: unexpected type | expected: document")
    }
}

// Lee a los ganadores del sorteo a traves del socket.
// Devuelve dichos ganadores como una lista de strings (los documentos)
// 
// En caso de error sera devuelto como segundo elemento
func (p *NationalLotteryCenter) ReadWinners() ([]string, error) {
    amount_d, err := readAll(p.conn, 4) // leer el int: cantidad de ganadores
    if err != nil {
        return []string{}, err
    }
    amount := int(binary.BigEndian.Uint32(amount_d))

    winners := []string{}

    for total_readed := 0; total_readed < amount; {
        document, err := p.ReadDocument()
        if err != nil {
            return []string{}, err
        }

        winners = append(winners, document)

        total_readed += 1
    }

    return winners, nil
}

// Realiza un poll hacia el servidor y se queda esperando la respuesta
// El tipo de respuesta sera devuelto como primer elemento
//  * INFO: hay ganadores
//  * WAIT: aun no se realizo el sorteo
//  * ERROR: ocurrio un error realizando el poll
// Si hay ganadores seran devueltos como el segundo elemento.
//
// En caso de error sera devuelto como tercer elemento
func (p *NationalLotteryCenter) PollWinners() (int, []string, error){
    poll_req := []byte{}
    poll_req = append(poll_req, POLL_TYPE)

    agency_id := make([]byte, 4)
    id, err := strconv.ParseInt(p.ID, 10, 32)
    if err != nil {
        return ERROR, []string{}, err
    }

    binary.BigEndian.PutUint32(agency_id, uint32(id))
    poll_req = append(poll_req, agency_id...)

    err = sendData(p.conn, poll_req)
    if err != nil {
        return ERROR, []string{}, err
    }

    status, err := readAll(p.conn, 1) // leer el tipo
    if err != nil {
        return ERROR, []string{}, err
    }

    if status[0] == byte(AWAIT_TYPE){
        return WAIT, []string{}, nil
    } else if status[0] == byte(WINNERS_TYPE){
        winners, err := p.ReadWinners()
        return INFO, winners, err
    } else {
        return ERROR, []string{}, errors.New("Poll error: unknown type")
    }
}