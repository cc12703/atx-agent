package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openatx/atx-agent/cmdctrl"
)

func buildScrcpyHandler() http.HandlerFunc {
	return singleFightNewerWebsocket(func(w http.ResponseWriter, r *http.Request, ws *websocket.Conn) {
		defer ws.Close()

		wsWrite := func(messageType int, data []byte) error {
			ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			return ws.WriteMessage(messageType, data)
		}

		if err := service.Start("scrcpy"); err != nil && err != cmdctrl.ErrAlreadyRunning {
			wsWrite(websocket.TextMessage, []byte("@scrcpy service start failed: "+err.Error()))
			return
		}

		waitServerStarted("scrcpy")

		//quitC := make(chan bool, 2)

		vconn, err := net.Dial("unix", scrcpySocketPath)
		if err != nil {
			log.Printf("dial for video %s error: %v", scrcpySocketPath, err)
			return
		}
		defer vconn.Close()

		cconn, err := net.Dial("unix", scrcpySocketPath)
		if err != nil {
			log.Printf("dial for control %s error: %v", scrcpySocketPath, err)
			return
		}
		defer cconn.Close()

		go func() {

			hbuf := make([]byte, 12)
			buf := make([]byte, SCRCPY_IOBUF_MAXSIZE)
			for {
				_, err := vconn.Read(hbuf)
				if err != nil {
					log.Println("read header err:", err)
					break
				}
				pkgSize := binary.BigEndian.Uint32(hbuf[8:12])
				if pkgSize > SCRCPY_IOBUF_MAXSIZE {
					log.Printf("pkgSize too big: %d", pkgSize)
				}
				rSize, err := io.ReadFull(vconn, buf[0:pkgSize])
				if err != nil {
					log.Printf("read body err: %v", err)
					break
				}
				err = wsWrite(websocket.BinaryMessage, buf[0:rSize])
				if err != nil {
					log.Printf("write websocket err: %v", err)
					if websocket.IsCloseError(err) {
						break
					}
				}
			}
		}()

		for {
			var msg = make(map[string]interface{})
			err = ws.ReadJSON(&msg)
			if err != nil {
				log.Println("readJson err:", err)
				if websocket.IsCloseError(err) {
					break
				}
				continue
			}

			err = nil
			if msg["type"] == "touch" {
				log.Println(msg)
				err = sendTouchRequest(cconn, msg)
			}

			if err != nil {
				log.Println("sendTouchRequest err:", err)
				if websocket.IsCloseError(err) {
					break
				}
			}

		}

	})

}

var OPER_TO_ACTIONS = map[string]uint8{
	"down":   0,
	"up":     1,
	"move":   2,
	"cancel": 3,
}

func valToInt(val interface{}) float64 {
	return math.Ceil(val.(float64))
}

func sendTouchRequest(conn net.Conn, reqData map[string]interface{}) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(2)                                    // type
	action := OPER_TO_ACTIONS[reqData["oper"].(string)] // action
	log.Printf("action: %d", action)
	buf.WriteByte(action) //action

	binary.Write(buf, binary.BigEndian, uint64(valToInt(reqData["id"]))) // pointerId

	binary.Write(buf, binary.BigEndian, uint32(valToInt(reqData["x"]))) // x
	binary.Write(buf, binary.BigEndian, uint32(valToInt(reqData["y"]))) // y

	binary.Write(buf, binary.BigEndian, uint16(valToInt(reqData["scrnWidth"])))  // scrnWidth
	binary.Write(buf, binary.BigEndian, uint16(valToInt(reqData["scrnHeight"]))) // scrnHeight

	binary.Write(buf, binary.BigEndian, uint16(valToInt(reqData["pressure"]))) // pressure
	binary.Write(buf, binary.BigEndian, uint32(0))                             // actionButton
	binary.Write(buf, binary.BigEndian, uint32(0))                             // buttons

	_, err := conn.Write(buf.Bytes())
	return err
}
