package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"

	"github.com/gorilla/websocket"
)

func main() {
	runtime.GOMAXPROCS(4)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	connections := make(chan *websocket.Conn, 3)
	messages := make(chan string)
	go viewPairs("bnbbtc@depth", connections, messages)
	go viewPairs("ethbtc@depth", connections, messages)
	go viewPairs("eosbtc@depth", connections, messages)

	var prevMsg string
	log.Println(`Ждем сообщение:`)
	for {

		select {
		case msg := <-messages:
			if prevMsg != msg {
				log.Println(msg)
			}
			prevMsg = msg
		case <-interrupt:
			i := 0
			for {
				c, ok := <-connections
				if ok == false {
					return
				}
				i = i + 1
				log.Println("Closing connection - ", i)
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					log.Println(" write close:", err)
				}
				c.Close()
				if i == 3 {
					close(connections)
					close(messages)
					return
				}

			}
			return
		}
	}
}
func viewPairs(tp string, conns chan *websocket.Conn, messages chan string) {
	c, _, err := websocket.DefaultDialer.Dial(`wss://stream.binance.com:9443/ws/`+tp, nil)
	conns <- c
	if err != nil {
		log.Fatal(tp, err)
	}
	done := make(chan struct{})

	go func() {
		defer close(done)
		j := 0
		for {
			j = j + 1
			var raw json.RawMessage
			err := c.ReadJSON(&raw)
			if err != nil {
				log.Println(tp, err)
				return
			}
			var content Content
			err1 := json.Unmarshal(raw, &content)
			if err1 != nil {
				log.Println(err1)
			}
			if len(content.B) > 0 && len(content.A) > 0 {
				max := findMax(content.B, 0)
				min := findMin(content.A, 0)
				mess := tp + " : " + `{bid: {price:` + max[0] + `, amount:` + max[1] + `}, ask:{{price:` + min[0] + `, amount:` + min[1] + `}}}`
				messages <- mess
			}

		}
	}()
}
func findMax(arr [][]string, i int32) []string {

	var max []string
	max = arr[0]
	maxVal, _ := strconv.ParseFloat(max[i], 32)

	for _, v := range arr {
		var val float64
		val, _ = strconv.ParseFloat(v[i], 32)
		if maxVal < val {
			max = v
			maxVal = val
		}
	}
	return max
}

func findMin(arr [][]string, i int32) []string {
	var min []string
	min = arr[0]
	minVal, _ := strconv.ParseFloat(min[i], 32)
	for _, v := range arr {
		var val float64
		val, _ = strconv.ParseFloat(v[i], 32)
		if minVal > val {
			min = v
			minVal = val
		}
	}
	return min
}

type Content struct {
	E  string     `json:"e"`
	E2 int64      `json:"E"`
	S  string     `json:"s"`
	U  int        `json:"U"`
	U2 int        `json:"u"`
	B  [][]string `json:"b"`
	A  [][]string `json:"a"`
}
