package common

import (
    "io"
    "os"
    "encoding/csv"
    "time"

    log "github.com/sirupsen/logrus"
)

// ClientConfig Configuracion usada por el cliente
type ClientConfig struct {
    ID            string
    ServerAddress string
    BetsFile      string
    BatchSize     uint
}

// Client entidad que lo encapsula
type Client struct {
    config ClientConfig
    center *NationalLotteryCenter
}

// NewClient inicializa un nuevo cliente, recibiendo la
// configuracion como parametro
func NewClient(config ClientConfig) *Client {
    client := &Client{
        config: config,
    }
    return client
}

// Run realiza la logica del cliente
// Primero recorre el archivo de apuestas y
//  envia mediante chunks las apuestas al servidor
// Luego, una vez terminado se empieza a realizar
//  el poll al servidor para obtener los ganadores
func (c *Client) Run () {
    err := c.StartClientLoop()
    if err != nil {
        log.Fatalf("action: client_loop | result: fail | error: %v", err)
        return
    }

    err = c.CheckWinners()
    if err != nil {
        log.Fatalf("action: check_winners | result: fail | error: %v", err)
        return
    }
}

// CheckWinners es la funcion que hace loop realizando
//  poll al servidor hasta obtener los ganadores
//
// Segun la logica propuesta, se genera una conexion
//  con el servidor, este puede contestar:
//  * Aun no se encuentra hecho el sorteo, en dicho caso
//      se frena la ejecución durante un tiempo determinado por
//      un exponential backoff segun cuantas veces se haya
//      rechazado la solicitud desde el servidor
//  * Ya se encuentra hecho el sorteo, en dicho caso
//      se reciben los documentos de los ganadores del sorteo. 
func (c *Client) CheckWinners() error {
    log.Infof("action: consulta_ganadores | result: starting")

    waitingTime := 1
    for {
        c.center = nil
        c.center = NewNationalLotteryCenter(c.config.ID, c.config.ServerAddress)

        log.Infof("action: polling | result: in_progress")
        status, winners, err := c.center.PollWinners()

        if err != nil {
            log.Fatalf("action: polling | result: fail | error: %v", err)
            c.center.Close()
            return err
        }
        log.Infof("action: polling | result: success")

        if status == WAIT {
            c.center.Close()
            log.Infof("action: consulta_ganadores | result: in_progress | sleeping time: %v", waitingTime)
            time.Sleep(time.Duration(waitingTime) * time.Second)
            waitingTime *= 2
        } else {
            log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", len(winners))
            c.center.Close()
            break
        }
    }
    return nil
}

// StartClientLoop es la funcion que lee el archivo y envia
//  utilizando chunks las apuestas al servidor.
// Se genera una conexion con el servidor y una vez establecida
//  se comienza a leer el archivo csv completando los llamados chunks
//  que no son mas que tiras de apuestas que se envian en conjunto
func (c *Client) StartClientLoop() error {
    c.center = NewNationalLotteryCenter(c.config.ID, c.config.ServerAddress)

    defer c.center.Close()

    file, err := os.Open(c.config.BetsFile)
    if err != nil {
        log.Fatalf("action: abrir_archivo | result: fail | file_name: %v | error: %v", c.config.BetsFile, err)
        return err
    }

    defer file.Close()

    reader := csv.NewReader(file)
    reader.Comma = ','
    reader.FieldsPerRecord = 5

    batch := make([]Bet, 0)
    i := 0
    for {
        // Read one record from csv
        record, err := reader.Read()
        if err == io.EOF {
            err = c.center.sendBatch(batch)
            if err != nil {
                return err
            }

            err = c.center.waitConfirmation()
            if err != nil {
                log.Fatalf("action: wait_confirmation | result: fail | error: %v", err)
                return err
            }

            i++
            log.Infof("action: batch enviado | result: success | no: %v | info: ultimo", i)

            batch = nil
            break
        }

        if err != nil {
            log.Fatalf("action: read_record | result: fail | error: %v", err)
            // closes in defer
            return err
        }

        batch = append(batch, fromRecord(record))

        if uint(len(batch)) == c.config.BatchSize {
            c.center.sendBatch(batch)
            if err != nil {
                return err
            }

            c.center.waitConfirmation()
            if err != nil {
                log.Fatalf("action: wait_confirmation | result: fail | error: %v", err)
                return err
            }

            i++
            log.Infof("action: batch enviado | result: success | no: %v", i)

            batch = nil
            continue
        }
    }

    err = c.center.Finish()
    if err != nil {
        log.Fatalf("action: finishing_connection | result: fail | error: %v", err)
    }

    return err
}