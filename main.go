package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/grandcat/zeroconf"
)

const instanceBase = "mydrop-"

var port = flag.Int("port", 4000, "Port for the airpipe process")
var iP = flag.String("ip", "127.0.0.1", "IP for the airpipe process")
var host = flag.String("host of the service", "godrop.local", "host")
var uid = flag.String("uid", "instance", "Your instance name")
var remote = flag.String("remote", "", "The remote process uid")

func main() {
	// parse command line flags
	flag.Parse()
	// register this process via mdns
	mdnsServer := register(*uid, *port)
	defer mdnsServer.Shutdown()

	server := Server{
		Addr: ":" + strconv.Itoa(*port),
	}

	// Clean exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-start(readStdin(), server.ListenForSingleConnection()):
	case <-sig:
		// Exit by user
	case <-time.After(time.Second * 120):
		// Exit by timeout
	}
}

func start(stdin <-chan []byte, conn <-chan net.Conn) chan bool {
	done := make(chan bool)

	go func(done chan bool) {
		select {
		case b := <-stdin:
			sendTo(b, *remote)
			done <- true
		case c := <-conn:
			handleConnection(c)
			done <- true
		}
	}(done)

	return done
}

func sendTo(b []byte, instance string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote := discoverRemote(ctx, instance)
	conn := connect(remote)

	if conn == nil {
		log.Fatalf("Was not able to connect to %s\n", remote.Instance)
	}

	defer conn.Close()

	buf := bytes.NewBuffer(b)
	io.Copy(conn, buf)

}

func connect(service *zeroconf.ServiceEntry) net.Conn {
	port := strconv.Itoa(service.Port)

	if c, err := tryIps("tcp4", port, service.AddrIPv4); err == nil {
		return c
	}

	if c, err := tryIps("tcp6", port, service.AddrIPv6); err == nil {
		return c
	}

	return nil
}

func tryIps(network, port string, ips []net.IP) (c net.Conn, err error) {

	for _, ip := range ips {
		addr := net.JoinHostPort(ip.String(), port)
		c, err = net.Dial(network, addr)
		if err == nil {
			return
		}
	}

	return nil, fmt.Errorf("No connection error")
}

func discoverRemote(parentContext context.Context, instance string) *zeroconf.ServiceEntry {
	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry)
	var result *zeroconf.ServiceEntry
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			if entry.Instance == instanceBase+instance {
				result = entry
				cancel()
			}
		}
	}(entries)

	discover(ctx, entries)

	return result
}

func handleConnection(c net.Conn) {
	io.Copy(os.Stdout, c)
}

func readStdin() chan []byte {
	buf := new(bytes.Buffer)
	res := make(chan []byte)
	go func(b *bytes.Buffer) {
		io.Copy(buf, os.Stdin)
		res <- buf.Bytes()
	}(buf)

	return res
}

func register(instance string, port int) *zeroconf.Server {
	name := "mydrop-" + instance
	service := "_drop._tcp"
	domain := ".local"
	meta := []string{"txtv=0", "lo=1", "la=2"}

	server, err := zeroconf.Register(name, service, domain, port, meta, nil)

	if err != nil {
		panic(err)
	}

	return server
}

func discover(parentContext context.Context, entries chan *zeroconf.ServiceEntry) {
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	ctx, cancel := context.WithTimeout(parentContext, time.Second*15)
	defer cancel()
	err = resolver.Browse(ctx, "_drop._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()

}
