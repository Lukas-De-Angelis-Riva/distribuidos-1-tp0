package common

import (
    "io"
    "os"
    "encoding/csv"
    "time"

    log "github.com/sirupsen/logrus"
)

// ClientConfig Configuration used by the client
type ClientConfig struct {
    ID            string
    ServerAddress string
    BetsFile      string
    BatchSize     uint
}

// Client Entity that encapsulates how
type Client struct {
    config ClientConfig
    center *NationalLotteryCenter
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
    client := &Client{
        config: config,
    }
    return client
}

func (c *Client) Run () {
    err := c.StartClientLoop()
    if err != nil {
        return
    }

    err = c.CheckWinners()
    if err != nil {
        return
    }
}

func (c *Client) CheckWinners() error {
    log.Infof("action: consulta_ganadores | result: starting")

    waitingTime := 1
    for {
        c.center = nil
        c.center = NewNationalLotteryCenter(c.config.ID, c.config.ServerAddress)

        log.Infof("action: polling | result: in_progress")
        status, winners, err := c.center.PollWinners()

        if err != nil {
            // log mal ahi
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

// StartClientLoop Send messages to the client until some time threshold is met
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