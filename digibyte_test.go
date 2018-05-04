package digibyte

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

	SetFee(fee float64) (bool, error)
	CreateAddress() (string, error)
	GetBalance() (float64, error)
	GetWalletInfo() (*BtcWalletInfo, error)
	GetBalanceByAddress(address string) (float64, error)
	SendToAddress(address string, amount float64) (string, error)
	GetTransaction(txid string) (map[string]interface{}, error)
	CheckTransaction(txid string) (bool, error)

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

type BtcWalletInfo struct {
	Hdmasterkeyid         string  `json:"hdmasterkeyid"`
	Walletname            string  `json:"walletname"`
	Walletversion         string  `json:"walletversion"`
	Balance               float64 `json:"balance"`
	UnconfirmedBalance    float64 `json:"unconfirmed_balance"`
	Keypoololdest         float64 `json:"keypoololdest"`
	Keypoolsize           int64   `json:"keypoolsize"`
	ImmatureBalance       float64 `json:"immature_balance"`
	Txcount               int64   `json:"txcount"`
	KeypoolsizeHdInternal int64   `json:"keypoolsize_hd_internal"`
	Paytxfee              float64 `json:"paytxfee"`
}

func (e BtcError) Error() string {
	return strconv.FormatInt(e.Code, 10) + ": " + e.Message
}

func (e *BtcWalletInfo) fill(obj map[string]interface{}) error {
	fields := map[string]string{
		"hdmasterkeyid":           "Hdmasterkeyid",
		"walletname":              "Walletname",
		"walletversion":           "Walletversion",
		"balance":                 "Balance",
		"unconfirmed_balance":     "UnconfirmedBalance",
		"keypoololdest":           "Keypoololdest",
		"keypoolsize":             "Keypoolsize",
		"immature_balance":        "ImmatureBalance",
		"txcount":                 "Txcount",
		"keypoolsize_hd_internal": "KeypoolsizeHdInternal",
		"paytxfee":                "Paytxfee",
	}
	for k, v := range fields {
		if val, ok := obj[k]; ok {
			field := reflect.ValueOf(e).Elem().FieldByName(v)
			if !field.IsValid() {
				continue
			}
			valtype := reflect.TypeOf(val).Name()
			switch field.Type().Name() {
			case "string":
				valueString := ""
				if valtype == "string" {
					valueString, _ = val.(string)
				} else {
					valueString = strconv.FormatFloat(val.(float64), 'f', -1, 64)
				}
				field.SetString(valueString)
			case "float64":
				var valueFloat float64
				if valtype == "string" {
					valueFloat, _ = strconv.ParseFloat(val.(string), 64)
				} else {
					valueFloat = val.(float64)
				}
				field.SetFloat(valueFloat)
			case "int64":
				var valueInt int64
				if valtype == "string" {
					valueInt, _ = strconv.ParseInt(val.(string), 10, 64)
				} else {
					valueInt = int64(val.(float64))
				}
				field.SetInt(valueInt)
			}
		}
	}

	return nil
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

func (b *btcClient) GetBalance() (float64, error) {
	req, e := json.Marshal(RequestData{Jsonrpc: "2.0", Method: "getbalance"})
	if e != nil {
		return 0, e
	}

	resp, err := b.sendRequest(req)
	if err != nil {
		return 0, err
	}
	if val, ok := resp["result"]; ok {
		var res float64
		if reflect.TypeOf(val).Name() == "string" {
			res, _ = strconv.ParseFloat(val.(string), 64)
		} else {
			res = val.(float64)
		}
		return res, nil
	}

	return 0, BtcError{Code: 500, Message: "No result"}
}

func (b *btcClient) GetWalletInfo() (*BtcWalletInfo, error) {
	req, e := json.Marshal(RequestData{Jsonrpc: "2.0", Method: "getwalletinfo"})
	if e != nil {
		return nil, e
	}

	resp, err := b.sendRequest(req)
	if err != nil {
		return nil, err
	}
	if val, ok := resp["result"]; ok {
		if result, ok := val.(map[string]interface{}); ok {
			res_obj := new(BtcWalletInfo)
			res_obj.fill(result)
			return res_obj, nil
		}
	}

	return nil, BtcError{Code: 500, Message: "No result"}
}


func (b *btcClient) CheckTransaction(txid string) (bool, error) {
	params := []interface{}{txid}
	req, e := json.Marshal(RequestData{Jsonrpc: "2.0", Method: "gettransaction", Params: params})
	if e != nil {
		return false, e
	}

	resp, err := b.sendRequest(req)
	if err != nil {
		return false, err
	}
	if val, ok := resp["result"]; ok {
		if result, ok := val.(map[string]interface{}); ok {
			//log.Println("transaction result", result)
			if val, ok := result["confirmations"]; ok {
				var res int64
				if reflect.TypeOf(val).Name() == "string" {
					res, _ = strconv.ParseInt(val.(string), 10, 64)
				} else {
					res = int64(val.(float64))
				}
				return res >= b.Confirmations, nil
			}
		}
	}

	return false, BtcError{Code: 500, Message: "No result"}
}


func (b *btcClient) GetBalanceByAddress(address string) (float64, error) {
	type BalanceRequestData struct {
		Jsonrpc string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}
	params := []interface{}{}
	params = append(params, address, b.Confirmations)
	req, e := json.Marshal(BalanceRequestData{Jsonrpc: "2.0", Method: "getreceivedbyaddress", Params: params})
	if e != nil {
		return 0, e
	}

	resp, err := b.sendRequest(req)
	if err != nil {
		return 0, err
	}
	if val, ok := resp["result"]; ok {
		var res float64
		if reflect.TypeOf(val).Name() == "string" {
			res, _ = strconv.ParseFloat(val.(string), 64)
		} else {
			res = val.(float64)
		}
		return res, nil
	}

	return 0, BtcError{Code: 500, Message: "No result"}
}

func (b *btcClient) SendToAddress(address string, amount float64) (string, error) {
	params := []interface{}{address, strconv.FormatFloat(amount, 'f', -1, 64)}
	req, e := json.Marshal(RequestData{Jsonrpc: "2.0", Method: "sendtoaddress", Params: params})
	if e != nil {
		return "", e
	}
	//log.Println("request", string(req))
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

func (b *btcClient) GetTransaction(txid string) (map[string]interface{}, error) {
	params := []interface{}{txid}
	req, e := json.Marshal(RequestData{Jsonrpc: "2.0", Method: "gettransaction", Params: params})
	if e != nil {
		return nil, e
	}

	resp, err := b.sendRequest(req)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}
	if val, ok := resp["result"]; ok {
		result = val.(map[string]interface{})
	}
	return result, nil
}

func (b *btcClient) SetFee(fee float64) (bool, error) {
	params := []interface{}{fee}
	req, e := json.Marshal(RequestData{
		Jsonrpc: "2.0",
		Method:  "settxfee",
		Params:  params,
	})
	if e != nil {
		return false, e
	}
	//log.Println("request", string(req))

	resp, err := b.sendRequest(req)
	if err != nil {
		return false, err
	}
	if val, ok := resp["result"]; ok {
		if res, ok := val.(bool); ok {
			return res, nil
		}
	}

	return false, BtcError{Code: 500, Message: "No result"}
}


var DigiByte = NewClient("http://a:b@localhost:14022/",6);

func TestCreateAddress(t *testing.T) {

	resp, err := DigiByte.CreateAddress()
	if err != nil {
		 t.Errorf("createAddress error: %+v", err)
		 t.FailNow()
	}
	t.Logf("createAddress result: %v", resp)
}

func TestGetBalance(t *testing.T) {

  resp, err := DigiByte.GetBalance()
  if err != nil {
     t.Errorf("getBalance error: %+v", err)
     t.FailNow()
  }
  t.Logf("getBalance result: %v", resp)
}

func TestGetWalletInfo(t *testing.T) {
  resp, err := DigiByte.GetWalletInfo()
  if err != nil {
     t.Errorf("getWalletInfo error: %+v", err)
     t.FailNow()
  }
  t.Logf("getWalletInfo result: %+v", resp)
}

func TestGetBalanceByAddress(t *testing.T) {
  resp, err := DigiByte.GetBalanceByAddress("DNTpSCCtoUExDZocboGYw9LAdQxedSm11n")
  if err != nil {
     t.Errorf("getBalanceByAddress error: %+v", err)
     t.FailNow()
  }
  t.Logf("getBalanceByAddress result: %v", resp)
}

func TestSendToAddress(t *testing.T) {

  addr := "DNTpSCCtoUExDZocboGYw9LAdQxedSm11n"
  resp, err := DigiByte.SendToAddress(addr, 0.00001)
  if err != nil {
     t.Errorf("sendToAddress error: %+v", err)
     t.FailNow()
  }
  t.Logf("sendToAddress result: %v", resp)
}

func TestGetTransaction(t *testing.T) {
  resp, err := DigiByte.GetTransaction("df5cae215feb9f135b94466dcd937ee303f95cdc095cc0aa688d86e3ec25f21b")
  if err != nil {
     t.Errorf("getTransaction error: %+v", err)
     t.FailNow()
  }
  t.Logf("getTransaction result: %v", resp)
}

func TestCheckTransaction(t *testing.T) {
  resp, err := DigiByte.CheckTransaction("df5cae215feb9f135b94466dcd937ee303f95cdc095cc0aa688d86e3ec25f21b")
  if err != nil {
     t.Errorf("getTransaction error: %+v", err)
     t.FailNow()
  }
  t.Logf("getTransaction result: %v", resp)
}

func TestSetFee(t *testing.T) {
  resp, err := DigiByte.SetFee(40 * 1024 * 0.00000001)
  if err != nil {
     t.Errorf("setFee error: %+v", err)
     t.FailNow()
  }
  t.Logf("setFee result: %v", resp)
}
