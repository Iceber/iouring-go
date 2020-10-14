# What is io_uring
[io_uring](http://kernel.dk/io_uring.pdf) 

[io_uring-wahtsnew](https://kernel.dk/io_uring-whatsnew.pdf) 

[LWN io_uring](https://lwn.net/Kernel/Index/#io_uring) 

[Lord of the io_uring](https://unixism.net/loti/)

# Features
- [x] register a file set for io_uring instance
- [x] support file IO
- [x] support socket IO
- [x] support IO timeout
- [x] link request
- [x] set timer
- [x] add request extra info, could get it from the result
- [ ] set logger
- [ ] register buffers and IO with buffers
- [ ] support Eventfd 
- [ ] support SQPoll 

# OS Requirements
* Linux Kernel >= 5.6

# Installation
```
go get github.com/iceber/iouring-go
```

# Quickstart
```
package main

import (
        "fmt"
        "os"

        "github.com/iceber/iouring-go"
)

var str = "io with iouring"

func main() {
        iour, err := iouring.New(1)
        if err != nil {
                panic(fmt.Sprintf("new IOURing error: %v", err))
        }

        file, err := os.Create("./tmp.txt")
        if err != nil {
                panic(err)
        }

        ch := make(chan *iouring.Result, 1)

        request := iouring.Write(int(file.Fd()), []byte(str))
        if _, err := iour.SubmitRequest(request, ch); err != nil {
                panic(err)
        }

        result := <-ch
        i, err := result.ReturnInt()
        if err != nil {
                fmt.Println("write error: ", err)
                return
        }

        fmt.Printf("write byte: %d\n", i)
}
```

# Request With Extra Info
```
request := iouring.RequestWithInfo(iouring.Write(int(file.Fd()), []byte(str)), file.Name())

iour.SubmitRequest(request, ch)

result <- ch
info, ok := result.GetRequestInfo().(string)
```

# Submit multitude request

```
var offset uint64
buf1 := make([]byte, 1024)
request1 := iouring.Pread(fd, buf1, offset)

offset += 1024
buf2 := make([]byte, 1024)
request2 := iouring.Pread(fd, buf1, offset)

iour.SubmitRequests([]iouring.Request{request1, request2}, nil)
```
requests is concurrent execution

# Link request
```
var offset uint64
buf := make([]byte, 1024)
request1 := iouring.Pread(fd, buf1, offset)
request2 := iouring.Write(int(os.Stdout.Fd()), buf)

iour.SubmitLinkRequests([]iouring.Request{request1, request2}, nil)
```

# Examples
[cat](https://github.com/Iceber/iouring-go/tree/main/examples/cat)

[concurrent-cat](https://github.com/Iceber/iouring-go/tree/main/examples/concurrent-cat)

[cp](https://github.com/Iceber/iouring-go/tree/main/examples/cp)

[request-with-timeout](https://github.com/Iceber/iouring-go/tree/main/examples/timeout/request-with-timeout)

[link-request](https://github.com/Iceber/iouring-go/tree/main/examples/link)

[link-with-timeout](https://github.com/Iceber/iouring-go/tree/main/examples/timeout/link-with-timeout)

[timer](https://github.com/Iceber/iouring-go/tree/main/examples/timeout/timer)

[echo](https://github.com/Iceber/iouring-go/tree/main/examples/echo)

# TODO
* friendly error
* add tests
* arguments type (eg. int and int32)
* set logger
