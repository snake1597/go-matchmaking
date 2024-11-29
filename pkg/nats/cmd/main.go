package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nats-io/nats.go"
)

var (
	mode = flag.String("mode", "pub", "Input: pub, sub")
	sub1 *nats.Subscription
	// sub2 *nats.Subscription
	// sub3 *nats.Subscription
)

func init() {
	flag.Parse()
}

func main() {
	nc, err := nats.Connect("secret@127.0.0.1:4222")
	if err != nil {
		log.Fatal(err)
	}

	if *mode == "sub" {
		err = subscribe(nc)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err = publish(nc)
		if err != nil {
			log.Fatal(err)
		}
	}

	time.Sleep(1 * time.Second)
	err = nc.Flush()
	if err != nil {
		log.Fatal(err)
	}

	if *mode == "sub" {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c

		close()
	}

	// Close connection
	nc.Close()
}

func publish(nc *nats.Conn) (err error) {
	err = nc.Publish("foo.test.a", []byte("Hello World"))
	if err != nil {
		return err
	}

	return nil
}

func subscribe(nc *nats.Conn) (err error) {
	go func() {
		subj := "foo.test.a"
		sub1, err = nc.Subscribe(subj, func(m *nats.Msg) {
			fmt.Printf("Received a message from %s : %s\n", subj, string(m.Data))
		})
		if err != nil {
			log.Fatal(err)
		}
	}()

	// go func() {
	// 	subj := "foo.test"
	// 	sub2, err = nc.Subscribe(subj, func(m *nats.Msg) {
	// 		fmt.Printf("Received a message from %s : %s\n", subj, string(m.Data))
	// 	})
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	// go func() {
	// 	subj := "foo.>"
	// 	sub3, err = nc.Subscribe(subj, func(m *nats.Msg) {
	// 		fmt.Printf("Received a message from %s : %s\n", subj, string(m.Data))
	// 	})
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	return nil
}

func close() {
	sub1.Unsubscribe()
	// sub2.Unsubscribe()
	// sub3.Unsubscribe()
}
