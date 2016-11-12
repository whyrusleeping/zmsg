package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	cli "github.com/urfave/cli"
)

type Message struct {
	To        string
	Timestamp time.Time
	Content   string
	Val       float64
}

func getMyAddresses() ([]string, error) {
	req := &Request{
		Method: "z_listaddresses",
	}

	var out struct {
		Result []string
	}
	err := request(req, &out)
	if err != nil {
		return nil, err
	}

	return out.Result, nil
}

type TxDesc struct {
	Txid   string
	Amount float64
	Memo   string
}

// getReceivedForAddr returns all received messages for a given address
func getReceivedForAddr(addr string) ([]*Message, error) {
	req := &Request{
		Method: "z_listreceivedbyaddress",
		Params: []string{addr},
	}

	var out struct {
		Result []*TxDesc
	}

	err := request(req, &out)
	if err != nil {
		return nil, err
	}

	var msgs []*Message
	for _, txd := range out.Result {
		decmemo, err := hex.DecodeString(txd.Memo)
		if err != nil {
			return nil, err
		}

		if decmemo[0] > 0xf7 {
			fmt.Println("warning: incorrectly formatted message received (b[0] > 0xf7)")
			continue
		}

		tx, err := getTransaction(txd.Txid)
		if err != nil {
			return nil, err
		}

		msg := &Message{
			Timestamp: time.Unix(tx.Time, 0),
			Val:       txd.Amount,
			Content:   string(bytes.TrimRight(decmemo, "\x00")),
			To:        addr,
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}

// CheckMessages returns all messages that the local zcash daemon has received
func CheckMessages() ([]*Message, error) {
	addrs, err := getMyAddresses()
	if err != nil {
		return nil, err
	}

	var allmsgs []*Message
	for _, myaddr := range addrs {
		msgs, err := getReceivedForAddr(myaddr)
		if err != nil {
			return nil, err
		}

		allmsgs = append(allmsgs, msgs...)
	}

	return allmsgs, nil
}

var ErrNoAddresses = fmt.Errorf("no addresses to send message from! (create one with the zcash-cli)")

// SendMessage sends a message to a given zcash address using a shielded transaction.
// It returns the transaction ID.
func SendMessage(from, to, msg string, msgval float64) (string, error) {
	if from == "" {
		// if no from address is specified, use the first local address
		myaddrs, err := getMyAddresses()
		if err != nil {
			return "", err
		}

		if len(myaddrs) == 0 {
			return "", ErrNoAddresses
		}

		from = myaddrs[0]
		fmt.Printf("sending message from %s\n", from)
	}

	req := &Request{
		Method: "z_sendmany",
		Params: []interface{}{
			from, // first parameter is address to send from (where the ZEC comes from)
			[]interface{}{
				map[string]interface{}{
					"amount":  msgval,
					"address": to,
					"memo":    hex.EncodeToString([]byte(msg)),
				},
			},
		},
	}

	var out struct {
		Result string
	}
	err := request(req, &out)
	if err != nil {
		return "", err
	}

	opid := out.Result
	txid, err := WaitForOperation(opid)
	if err != nil {
		return "", err
	}

	return txid, nil
}

type opStatus struct {
	Id           string
	Status       string
	CreationTime uint64 `json:"creation_time"`
	Error        Error
	Result       struct {
		Txid string
	}
}

func checkOperationStatus(opid string) (*opStatus, error) {
	req := &Request{
		Method: "z_getoperationstatus",
		Params: []interface{}{[]string{opid}},
	}

	var out struct {
		Result []*opStatus
	}
	err := request(req, &out)
	if err != nil {
		return nil, err
	}

	return out.Result[0], nil
}

type Transaction struct {
	Time          int64
	Confirmations int
}

func getTransaction(txid string) (*Transaction, error) {
	req := &Request{
		Method: "gettransaction",
		Params: []string{txid},
	}

	var out struct {
		Result Transaction
	}

	err := request(req, &out)
	if err != nil {
		return nil, err
	}

	return &out.Result, nil
}

// WaitForOperation polls the operations status until it either fails or
// succeeds.
func WaitForOperation(opid string) (string, error) {
	i := 0
	for range time.Tick(time.Second) {
		status, err := checkOperationStatus(opid)
		if err != nil {
			return "", err
		}

		switch status.Status {
		case "failed":
			fmt.Println("operation failed!")
			fmt.Println("reason: ", status.Error.Message)
			return "", fmt.Errorf(status.Error.Message)
		case "executing":
			// in progress, print a progress thingy?
			fmt.Printf("\r                      \r")
			fmt.Print("sending message")
			for j := 0; j <= (i % 4); j++ {
				fmt.Print(".")
			}
		case "success":
			fmt.Println("\nMessage sent successfuly!")
			return status.Result.Txid, nil
		default:
			fmt.Printf("%#v\n", status)
		}
		i++
	}
	return "", nil
}

var CheckCmd = cli.Command{
	Name:  "check",
	Usage: "check for messages.",
	Action: func(c *cli.Context) error {
		msgs, err := CheckMessages()
		if err != nil {
			return err
		}

		div := strings.Repeat("=", 80)
		fmt.Println(div)
		fmt.Printf("> Got %d messages.\n", len(msgs))
		fmt.Println(div)
		for i, m := range msgs {
			fmt.Printf("| Message #%d (val = %f)\n", i, m.Val)
			fmt.Printf("| To: %s\n", m.To)
			fmt.Printf("| Date: %s\n|\n", m.Timestamp)
			fmt.Println("| ", strings.Replace(m.Content, "\n", "\n| ", -1))
			fmt.Println(div)
		}

		return nil
	},
}

var SendCmd = cli.Command{
	Name:  "sendmsg",
	Usage: "send a message to another zmsg user.",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "from",
			Usage: "address to send message from",
		},
		cli.StringFlag{
			Name:  "to",
			Usage: "address to send message to",
		},
		cli.Float64Flag{
			Name:  "txval",
			Value: 0.00001,
			Usage: "specify the amount of ZEC to send with messages.",
		},
	},
	Action: func(c *cli.Context) error {
		to := c.String("to")
		from := c.String("from")
		msg := strings.Join(c.Args(), " ")

		if to == "" {
			return fmt.Errorf("please specify an address to send the message to")
		}

		if msg == "" {
			return fmt.Errorf("no message specified")
		}
		fmt.Printf("message: %q\n", msg)

		txid, err := SendMessage(from, to, msg, c.Float64("txval"))
		if err != nil {
			return err
		}

		fmt.Printf("message sent! (txid = %s)\n", txid)
		return nil
	},
}

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Author = "whyrusleeping"
	app.Email = "why@ipfs.io"
	app.Usage = "send and receive zero knowledge messages"
	app.Commands = []cli.Command{
		CheckCmd,
		SendCmd,
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
