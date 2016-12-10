package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var DefaultUser string
var DefaultPass string

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}

type Response struct {
	Result interface{} `json:"result"`
	Error  Error       `json:"error"`
}

type Request struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

func Do(obj *Request, out interface{}) error {
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
	req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(DefaultUser+":"+DefaultPass)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return fmt.Errorf("failed to connect to zcash daemon, is it running?")
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var res Response
		err := json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			fmt.Println("error reading http body: ", err)
		}

		return res.Error
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
