package main

import (
   "fmt"
   "net/http"
   "github.com/gorilla/websocket"
)

var (
   upgrader = websocket.Upgrader{}
   connections []*websocket.Conn
)

type SignonMessage struct {
   Interval int
   Tree     []map[string][]string
}

func change(c *websocket.Conn) {
   enabled := []string{"enabled"}

   for _, sensor := range sensors {
      for _, elem := range sensor.enabled {
         enabled = append(enabled, elem.desc)
      }
   }

   c.WriteJSON(&enabled)
}

func update(samples *[]uint64) {
   for _, c := range connections {
      err := c.WriteJSON(samples)
      if err != nil && *debug {
         fmt.Println("failed writing: ", err)
      }
   }
}

func remove(c *websocket.Conn) {
   for i := range connections {
      if connections[i] == c {
         connections[i] = connections[len(connections)-1]
         connections = connections[:len(connections)-1]
         return
      }
   }

   panic("element not found")
}

func monitor(w http.ResponseWriter, r *http.Request) {
   c, err := upgrader.Upgrade(w, r, nil)
   if err != nil {
      if *debug {
         fmt.Print("upgrade:", err)
      }
      return
   }

   defer c.Close()

   // handshake
   _, message, err := c.ReadMessage()
   if err != nil {
      if *debug {
         fmt.Println("read:", err)
      }
      return
   }

   if string(message) != "463ba1974b06" {
      if *debug {
         fmt.Println("auth failed")
      }
      return
   }

   if *debug {
      fmt.Println("auth succeeded")
   }

   msg := SignonMessage{Interval: interval, Tree: make([]map[string][]string, len(sensors))}

   for i, sensor := range sensors {
      msg.Tree[i] = make(map[string][]string)
      name := sensor.sensor.name()
      supported := *sensor.sensor.supported()

      msg.Tree[i][name] = make([]string, len(supported))

      for j, val := range supported {
         msg.Tree[i][name][j] = val.desc
      }
   }

   err = c.WriteJSON(&msg)
   if err != nil {
      if *debug {
         fmt.Println("write:", err)
      }
      return
   }

   change(c);
   connections = append(connections, c)

   for {
      var msg string
      err := c.ReadJSON(&msg)
      if err != nil {
         if *debug {
            fmt.Println("failed reading:", err)
         }
         remove(c)
         break
      }

      if *debug {
         fmt.Printf("recv %+v\n", msg)
      }
   }
}

func initweb(addr string) {
   fileServer := http.FileServer(http.Dir("resources"))
   http.Handle("/", fileServer)
   http.HandleFunc("/monitor", monitor)

   go http.ListenAndServe(addr, nil)
   fmt.Printf("interface available at http://%v\n", addr)
}
