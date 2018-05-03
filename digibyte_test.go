package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"time"
	"testing"
)

type Client interface {
	CreateAddress() (string, error)
}

type RequestData struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type btcClient struct {
	Url           string
	Confirmations int64
}

type BtcError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func (e BtcError) Error() string {
	return strconv.FormatInt(e.Code, 10) + ": " + e.Message
}

func checkBtcError(obj map[string]interface{}) error {
	if val, ok := obj["error"]; ok {
		if error, ok := val.(map[string]interface{}); ok {
			ecode := int64(0)
			emsg := ""
			if val, ok = error["code"]; ok {
				if reflect.TypeOf(val).Name() == "string" {
					ecode, _ = strconv.ParseInt(val.(string), 10, 64)
				} else {
					ecode = int64(val.(float64))
				}
			}
			if val, ok = error["message"]; ok {
				emsg = val.(string)
			}
			if ecode != 0 {
				res := BtcError{Code: ecode, Message: emsg}
				return res
			}
		}
	}
	return nil
}

func (b *btcClient) sendRequest(reqbody []byte) (map[string]interface{}, error) {
	req, err := http.NewRequest("POST", b.Url, bytes.NewBuffer(reqbody))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	var body []byte

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res_dat map[string]interface{}
	if err = json.Unmarshal(body, &res_dat); err != nil {
		return nil, err
	}
	//log.Println("response", res_dat)
	btce := checkBtcError(res_dat)
	if btce != nil {
		//log.Println("response error", btce)
		return nil, btce
	}

	return res_dat, nil
}

func NewClient(url string, confirmations int64) Client {
	return &btcClient{Url: url, Confirmations: confirmations}
}

func (b *btcClient) CreateAddress() (string, error) {
	req, e := json.Marshal(RequestData{Jsonrpc: "2.0", Method: "getnewaddress"})
	if e != nil {
		return "", e
	}

	resp, err := b.sendRequest(req)
	if err != nil {
		return "", err
	}
	if val, ok := resp["result"]; ok {
		if res, ok := val.(string); ok {
			return res, nil
		}
	}

	return "", BtcError{Code: 500, Message: "No result"}
}

func TestCreateAddress(t *testing.T) {

        var DigiByte = NewClient("http://a:b@localhost:14022/",6);

	resp, err := DigiByte.CreateAddress()
	if err != nil {
		 t.Errorf("createAddress error: %+v", err)
		 t.FailNow()
	}
	t.Logf("createAddress result: %v", resp)
}

