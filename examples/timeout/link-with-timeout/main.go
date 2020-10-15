package main

import (
	"fmt"
	"time"

	"github.com/iceber/iouring-go"
)

func main() {
	iour, err := iouring.New(10)
	if err != nil {
		panic(fmt.Sprintf("new IOURing error: %v", err))
	}
	defer iour.Close()

	now := time.Now()

	rs := iouring.RequestWithTimeout(iouring.Timeout(2*time.Second), 1*time.Second)
	rs1 := iouring.RequestWithTimeout(iouring.Timeout(5*time.Second), 4*time.Second)
	rs = append(rs, rs1...)

	ch := make(chan *iouring.Result, 1)
	if err := iour.SubmitLinkRequests(rs, ch); err != nil {
		panic(err)
	}

	for i := 0; i < 2; i++ {
		result := <-ch
		if err := result.Err(); err != nil {
			fmt.Println("error: ", err)
		}
		fmt.Println(time.Now().Sub(now))
	}
}
