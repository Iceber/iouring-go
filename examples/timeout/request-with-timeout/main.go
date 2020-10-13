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
	now := time.Now()

	ch := make(chan *iouring.Result, 1)
	request := iouring.Timeout(2 * time.Second)
	err = iour.SubmitRequests(iouring.RequestWithTimeout(request, 1*time.Second), ch)
	if err != nil {
		panic(err)
	}

	result := <-ch
	if err := result.Err(); err != nil {
		fmt.Println("error: ", err)
	}
	fmt.Println(time.Now().Sub(now))
}
