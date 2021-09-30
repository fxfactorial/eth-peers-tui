package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
	"github.com/oschwald/maxminddb-golang"
	"github.com/rivo/tview"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

type peerStuff struct {
	remoteAddr string
	location   string
	active     bool
	enode      string
}

type TableData struct {
	tview.TableContentReadOnly
	holdingPeer map[int]peerStuff
}

func (d *TableData) GetCell(row, column int) *tview.TableCell {
	peer, ok := d.holdingPeer[row]

	if !ok {
		fmt.Println("dont have it", row, column)
		return nil
	}

	if column == 0 {
		return tview.NewTableCell(peer.remoteAddr)
	}

	if column == 1 {
		return tview.NewTableCell(peer.location)
	}

	if column == 2 {
		active := peer.active
		if active {
			return tview.NewTableCell("[green]active")
		}

		return tview.NewTableCell("[red]inactive")
	}

	if column == 3 {
		return tview.NewTableCell(peer.enode[:20] + "...")
	}

	return nil

}

func (d *TableData) GetRowCount() int {
	return len(d.holdingPeer)
}

func (d *TableData) GetColumnCount() int {
	return 4
}

func (d *TableData) addPeer(ip, remoteAddr, location, enode string) {
	p := peerStuff{
		remoteAddr: remoteAddr,
		location:   location,
		active:     true,
		enode:      enode,
	}

	if len(d.holdingPeer) == 0 {
		d.holdingPeer[0] = p
	} else {
		d.holdingPeer[len(d.holdingPeer)] = p
	}

}

func main() {
	flag.Parse()
	log.SetFlags(0)

	ipDB, err := maxminddb.Open("GeoLite2-City_20210928/GeoLite2-City.mmdb")

	if err != nil {
		fmt.Println("ip db error ", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		log.Fatal("dial:", err)
	}

	defer c.Close()

	done := make(chan struct{})
	app := tview.NewApplication()
	data := &TableData{
		holdingPeer: map[int]peerStuff{},
	}

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, true).
		SetContent(data)
		//		SetTitle("eth p2p nodes").
		//		SetTitleAlign(1).
		//		SetTitleColor(tcell.Color105)

	go func() {
		if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
			panic(err)
		}
	}()

	go func() {
		defer close(done)
		buffer := map[string]interface{}{}

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			if err := json.Unmarshal(message, &buffer); err != nil {
				fmt.Println("talk about it", err)
			}

			ipStr := buffer["plain-ip"].(string)
			ip := net.ParseIP(ipStr)
			remoteStr := buffer["remote"].(string)
			enode := buffer["enode"].(string)

			var record struct {
				City struct {
					Names struct {
						En string `maxminddb:"en"`
					} `maxminddb:"names"`
				} `maxminddb:"city"`
				Country struct {
					ISOCode string `maxminddb:"iso_code"`
				} `maxminddb:"country"`
			} // Or any appropriate struct

			if err := ipDB.Lookup(ip, &record); err != nil {
				fmt.Println("issue on lookup", err)
				return
			}

			//			fmt.Println("here is record loop?", record)
			//			list.AddItem(ipStr, buffer["remote"].(string), 'e', nil)
			data.addPeer(
				ipStr, remoteStr,
				fmt.Sprintf("%s:%s", record.Country.ISOCode, record.City.Names.En),
				enode,
			)
			app.Draw()
		}
	}()

	err = c.WriteMessage(websocket.TextMessage, []byte("subscribe to peer stuff please"))

	if err != nil {
		log.Println("write:", err)
		return
	}

	<-interrupt
	log.Println("interrupt")

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	//	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	if err != nil {
		log.Println("write close:", err)
		return
	}

}
