package main

import (
	"fmt"
	"time"

	"github.com/iceber/iouring-go"
)

func main() {
	iour, err := iouring.New(3)
	if err != nil {
		panic(fmt.Sprintf("new IOURing error: %v", err))
	}
	now := time.Now()

	request2 := iouring.Timeout(2 * time.Second)
	request1 := iouring.Timeout(5 * time.Second)
	ch := make(chan *iouring.Result, 1)
	if err := iour.SubmitRequests([]iouring.Request{request1, request2}, ch); err != nil {
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
