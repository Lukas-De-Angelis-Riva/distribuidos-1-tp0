package common

import (

	log "github.com/sirupsen/logrus"
)

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	Name          string
	Surname       string
	Document      string
	BirthDate     string
	Number        string
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

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	c.center = NewNationalLotteryCenter(c.config.ID, c.config.ServerAddress)

	bet := Bet{
		Name: c.config.Name,
		Surname: c.config.Surname,
		Document: c.config.Document,
		BirthDate: c.config.BirthDate,
		Number: c.config.Number,
	}

	err := c.center.sendBet(bet)
	if err != nil {
        log.Fatalf(
            "action: apuesta_enviada | result: fail | client_id: %v | error: %v",
            c.config.ID,
            err,
        )
        c.center.Close()
        return
	}

	err = c.center.waitConfirmation(bet)
	if err != nil {
        log.Fatalf(
            "action: esperando_confirmacion | result: fail | client_id: %v | error: %v",
            c.config.ID,
            err,
        )
        c.center.Close()
        return
	}

	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", bet.Document, bet.Number)

	c.center.Close()
	log.Info("action: closing_socket | result success")
}