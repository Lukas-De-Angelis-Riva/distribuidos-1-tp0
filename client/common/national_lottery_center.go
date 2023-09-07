package common

import(
    "net"
    "encoding/binary"
    "errors"

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

const FINISH_TYPE = 'F'
const OK_TYPE = 'O'

type NationalLotteryCenter struct {
    conn net.Conn 
    ID string
}

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

func serializeString(field_type byte, field string) []byte {
    serialized := []byte{}
    serialized = append(serialized, field_type)

    length := make([]byte, 4)
    binary.BigEndian.PutUint32(length, uint32(len(field)))
    serialized = append(serialized, length...)

    return append(serialized, []byte(field)...)
}

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

func (p *NationalLotteryCenter) sendBet(bet Bet) error {
    data := p.serializeBet(bet)
    err := sendData(p.conn, data)
    if err != nil {
        log.Errorf("action: send_message | result: fail | client_id: %v | error: %v", p.ID, err)
        return err
    }

    return nil
}


// OJO NO HACE LO QUE SE PIDE; TIENE QUE SER SOLO UN LLAMADO A SEND
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


func (p *NationalLotteryCenter) Finish() error {
    goodbye := []byte{}
    goodbye = append(goodbye, FINISH_TYPE)

    err := sendData(p.conn, goodbye)
    return err

}

func (p *NationalLotteryCenter) Close() {
    p.conn.Close()
}