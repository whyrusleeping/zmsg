package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	cli "github.com/urfave/cli"
)

var username string
var password string

func init() {
	u, p, err := readAuthCreds()
	if err != nil {
		fmt.Println("Error reading zcash config: ", err)
	}

	username = u
	password = p
}

func readAuthCreds() (string, string, error) {
	homedir := os.Getenv("HOME")
	confpath := filepath.Join(homedir, ".zcash/zcash.conf")
	fi, err := os.Open(confpath)
	if err != nil {
		return "", "", err
	}
	defer fi.Close()

	var user string
	var pass string
	scan := bufio.NewScanner(fi)
	for scan.Scan() {
		parts := strings.SplitN(scan.Text(), "=", 2)
		if len(parts) < 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "rpcuser":
			user = val
		case "rpcpassword":
			pass = val
		}
	}

	return user, pass, nil
}

type Message struct {
	From    string
	To      string
	Content string
	Val     float64
}

func getMyAddresses() ([]string, error) {
	req := map[string]interface{}{
		"method": "z_listaddresses",
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

type txDesc struct {
	Txid   string
	Amount float64
	Memo   string
}

func getReceivedForAddr(addr string) ([]*Message, error) {
	req := map[string]interface{}{
		"method": "z_listreceivedbyaddress",
		"params": []string{addr},
	}

	var out struct {
		Result []*txDesc
	}

	err := request(req, &out)
	if err != nil {
		return nil, err
	}

	var msgs []*Message
	for _, tx := range out.Result {
		decmemo, err := hex.DecodeString(tx.Memo)
		if err != nil {
			return nil, err
		}

		msg := &Message{
			Val:     tx.Amount,
			Content: string(bytes.TrimRight(decmemo, "\x00")),
			To:      addr,
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}

func checkMessages() ([]*Message, error) {
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

func request(obj interface{}, out interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	body := bytes.NewReader(data)

	req, err := http.NewRequest("POST", "http://localhost:8232/", body)
	if err != nil {
		return err
	}

	// auth auth baby
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("error reading http body: ", err)
		}

		fmt.Println("Error from rpc server: ", string(data))

		return fmt.Errorf("http %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func sendMessage(from, to, msg string) (string, error) {
	// {"method":"z_sendmany","params":["",[{"amount":0.00100000,"address":"","memo":""}]],"id":1}
	if from == "" {
		myaddrs, err := getMyAddresses()
		if err != nil {
			return "", err
		}

		if len(myaddrs) == 0 {
			return "", fmt.Errorf("no addresses to send message from! (create one with the zcash-cli)")
		}

		from = myaddrs[0]
		fmt.Printf("sending message from %s\n", from)
	}

	req := map[string]interface{}{
		"method": "z_sendmany",
		"params": []interface{}{
			from,
			[]interface{}{
				map[string]interface{}{
					"amount":  0.00001,
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
	err = waitForOperation(opid)
	if err != nil {
		return "", err
	}

	return opid, nil
}

type opStatus struct {
	Id           string
	Status       string
	CreationTime uint64 `json:"creation_time"`
	Error        struct {
		Code    int
		Message string
	}
}

func checkOperationStatus(opid string) (*opStatus, error) {
	req := map[string]interface{}{
		"method": "z_getoperationstatus",
		"params": []interface{}{[]string{opid}},
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

func waitForOperation(opid string) error {
	i := 0
	for range time.Tick(time.Second) {
		status, err := checkOperationStatus(opid)
		if err != nil {
			return err
		}

		switch status.Status {
		case "failed":
			fmt.Println("operation failed!")
			fmt.Println("reason: ", status.Error.Message)
			return fmt.Errorf(status.Error.Message)
		case "executing":
			// in progress, print a progress thingy?
			fmt.Printf("\r                      ")
			fmt.Print("sending message")
			for j := 0; j <= (i % 4); j++ {
				fmt.Print(".")
			}
		case "success":
			fmt.Println("\nMessage sent successfuly!")
			return nil
		default:
			fmt.Printf("%#v\n", status)
		}
	}
	return nil
}

func main() {
	app := cli.NewApp()

	CheckCmd := cli.Command{
		Name: "check",
		Action: func(c *cli.Context) error {
			msgs, err := checkMessages()
			if err != nil {
				return err
			}

			for _, m := range msgs {
				fmt.Println("==========================================")
				fmt.Printf("Message (val = %f)\n", m.Val)
				fmt.Printf("From: %s\n", m.From)
				fmt.Printf("To: %s\n", m.To)
				fmt.Println(m.Content)
				fmt.Println("==========================================")
			}

			return nil
		},
	}

	SendCmd := cli.Command{
		Name: "sendmsg",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "from",
				Usage: "address to send message from",
			},
			cli.StringFlag{
				Name:  "to",
				Usage: "address to send message to",
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

			txid, err := sendMessage(from, to, msg)
			if err != nil {
				return err
			}

			fmt.Printf("message sent! (txid = %s)\n", txid)
			return nil
		},
	}

	app.Commands = []cli.Command{
		CheckCmd,
		SendCmd,
	}

	app.RunAndExitOnError()
}
