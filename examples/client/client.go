// Program client demonstrates how to set up a JSON-RPC 2.0 client using the
// bitbucket.org/creachadair/jrpc2 package.
//
// Usage (communicates with the server example):
//
//   go build bitbucket.org/creachadair/jrpc2/examples/client
//   ./client -server :8080
//
package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"sync"

	"bitbucket.org/creachadair/jrpc2"
)

var serverAddr = flag.String("server", "", "Server address")

var (
	// Reflective call wrappers for the remote methods.
	add  = jrpc2.NewCaller("Math.Add", int(0), int(0), jrpc2.Variadic()).(func(*jrpc2.Client, ...int) (int, error))
	div  = jrpc2.NewCaller("Math.Div", binarg{}, float64(0)).(func(*jrpc2.Client, binarg) (float64, error))
	stat = jrpc2.NewCaller("Math.Status", nil, "").(func(*jrpc2.Client) (string, error))
)

type binarg struct{ X, Y int }

func intResult(rsp *jrpc2.Response) int {
	var v int
	if err := rsp.UnmarshalResult(&v); err != nil {
		log.Fatalln("UnmarshalResult:", err)
	}
	return v
}

func main() {
	flag.Parse()
	if *serverAddr == "" {
		log.Fatal("You must provide -server address to connect to")
	}

	conn, err := net.Dial("tcp", *serverAddr)
	if err != nil {
		log.Fatalf("Dial %q: %v", *serverAddr, err)
	}
	log.Printf("Connected to %v", conn.RemoteAddr())

	// Start up the client, and enable logging to stderr.
	cli := jrpc2.NewClient(conn, nil)
	defer cli.Close()

	log.Print("\n-- Sending some individual requests...")
	if sum, err := add(cli, 1, 3, 5, 7); err != nil {
		log.Fatalln("Math.Add:", err)
	} else {
		log.Printf("Math.Add result=%d", sum)
	}
	if quot, err := div(cli, binarg{82, 19}); err != nil {
		log.Fatalln("Math.Div:", err)
	} else {
		log.Printf("Math.Div result=%.3f", quot)
	}
	if s, err := stat(cli); err != nil {
		log.Fatalln("Math.Status:", err)
	} else {
		log.Printf("Math.Status result=%q", s)
	}

	// An error condition (division by zero)
	if quot, err := div(cli, binarg{15, 0}); err != nil {
		log.Printf("Math.Div err=%v", err)
	} else {
		log.Fatalf("Math.Div succeeded unexpectedly: result=%v", quot)
	}

	log.Print("\n-- Sending a batch of requests...")
	var reqs []*jrpc2.Request
	for i := 1; i <= 5; i++ {
		x := rand.Intn(100)
		for j := 1; j <= 5; j++ {
			y := rand.Intn(100)
			req, err := cli.Req("Math.Mul", struct{ X, Y int }{x, y})
			if err != nil {
				log.Fatalf("Req (%d*%d): %v", x, y, err)
			}
			reqs = append(reqs, req)
		}
	}
	ps, err := cli.Send(reqs...)
	if err != nil {
		log.Fatalln("Call:", err)
	}
	for i, p := range ps {
		rsp, err := p.Wait()
		if err != nil {
			log.Printf("Req %q %s failed: %v", reqs[i].Method(), p.ID(), err)
			continue
		}
		log.Printf("Req %q %s: result=%d", reqs[i].Method(), rsp.ID(), intResult(rsp))
	}

	log.Print("\n-- Sending individual concurrent requests...")
	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		x := rand.Intn(100)
		for j := 1; j <= 5; j++ {
			y := rand.Intn(100)
			wg.Add(1)
			go func() {
				defer wg.Done()
				rsp, err := cli.Call("Math.Sub", struct{ X, Y int }{x, y})
				if err != nil {
					log.Printf("Req (%d-%d) failed: %v", x, y, err)
					return
				}
				log.Printf("Req (%d-%d): result=%d", x, y, intResult(rsp))
			}()
		}
	}
	wg.Wait()

	log.Print("\n-- Sending a notification...")
	if err := cli.Notify("Post.Alert", struct{ Msg string }{"There is a fire!"}); err != nil {
		log.Fatalln("Notify:", err)
	}
}
