package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	. "github.com/IPoWS/node-core/data/nodes"
	"github.com/IPoWS/node-core/link"
	log "github.com/sirupsen/logrus"
)

var (
	nodes Nodes
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	Filemu.RLock()
	http.ServeFile(w, r, Nodesfile)
	Filemu.RUnlock()
}

// websocket实现
func nps(w http.ResponseWriter, r *http.Request) {
	// 检查是否GET请求
	if methodIs("GET", w, r) {
		// 检查uid
		q := r.URL.Query()
		ent := getFirst("ent", &q)
		if len(ent) == 6 {
			host := getIPPortStr(r)
			_, ok := nodes.Nodes[host]
			if ok {
				serveFile(w, r)
			} else {
				newip := uint32(time.Now().UnixNano()>>16) ^ rand.Uint32()
				for hasExist(newip) {
					newip = uint32(time.Now().UnixNano()>>16) ^ rand.Uint32()
				}
				wsips = append(wsips, newip)
				nip64 := uint64(newip)<<32 | 1
				wsip, _, err := link.UpgradeLink(w, r, nip64)
				if err == nil && wsip == nip64 {
					nodes.Nodes[host] = ent
					err = nodes.Save()
					if err == nil {
						serveFile(w, r)
					} else {
						http.Error(w, "Save node file error.", http.StatusInternalServerError)
					}
				} else {
					http.Error(w, "Init link error.", http.StatusBadRequest)
					log.Errorf("[/nps] wsip: %x, nip64: %x, err: %v.", wsip, nip64, err)
				}
			}
		} else {
			http.Error(w, "Invalid entry length.", http.StatusBadRequest)
			log.Errorln("[/nps] invalid entry length.")
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	l := len(os.Args)
	if l >= 2 && l <= 4 {
		listener, err := net.Listen("tcp", os.Args[1])
		if err != nil {
			panic(err)
		} else {
			if l == 4 {
				uid, err := strconv.Atoi(os.Args[3])
				if err == nil {
					syscall.Setuid(uid)
					syscall.Setgid(uid)
				} else {
					panic(err)
				}
			}
			if l >= 3 && os.Args[2] != "" {
				Nodesfile = os.Args[2]
			}
			nodes.Load()
			http.HandleFunc("/nps", nps)
			link.SetNPSUrl("ws://" + os.Args[1] + "/nps")
			link.InitEntry("npsent")
			go func() {
				time.Sleep(time.Second * 5)
				link.Register("npsent")
			}()
			log.Fatal(http.Serve(listener, nil))
		}
	} else {
		fmt.Println("Usage: host:port (nodesfile) (uid)")
	}
}
