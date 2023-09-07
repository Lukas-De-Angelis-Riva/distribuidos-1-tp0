package common

import(
    "net"
    "encoding/binary"
    "errors"

    log "github.com/sirupsen/logrus"
)

const BET_TYPE = 'B'
const AGENCY_NAME_TYPE = 'A'
const NAME_TYPE = 'N'
const LAST_NAME_TYPE = 'L'
const DOCUMENT_TYPE = 'D'
const BIRTHDATE_TYPE = 'H'
const NUMBER_TYPE = 'U'

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
    log.Debugf("action: send_message | result: success | encoded msg: %v", data)

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

func (p *NationalLotteryCenter) sendBet(bet Bet) error {
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

    err := sendData(p.conn, data)
    if err != nil {
        log.Errorf("action: send_message | result: fail | client_id: %v | error: %v", p.ID, err)
        return err
    }

    return nil
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

func (p *NationalLotteryCenter) waitConfirmation(bet Bet) error {
    _, err := readAll(p.conn, 1) // leer el tipo OK
    if err != nil {
        return errors.New("Confirmation error: cannot read OK_Type")
    }

    confirmation_len_d, err := readAll(p.conn, 4) // leer el largo uint32
    if err != nil {
        return errors.New("Confirmation error: cannot read length")
    }
    confirmation_len := int(binary.BigEndian.Uint32(confirmation_len_d))

    message_d, err := readAll(p.conn, confirmation_len) // leo el numero de apuesta
    if err != nil {
        return err
    }
    message := string(message_d)

    if(bet.Number != message){
        return errors.New("Confirmation error: Invalid number")
    }

    return nil
}

func (p *NationalLotteryCenter) Close() {
    p.conn.Close()
}