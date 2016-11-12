package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Error struct {
	Code    int
	Message string
}

type Response struct {
	Result interface{}
	Error  Error
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

		return fmt.Errorf(res.Error.Message)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
