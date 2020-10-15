package main

import (
	"fmt"
	"os"

	"github.com/iceber/iouring-go"
)

const blockSize int64 = 32 * 1024

var buffers [][]byte

func getBuffers(size int64) [][]byte {
	blocks := int(size / blockSize)
	if size%blockSize != 0 {
		blocks++
	}

	for i := 0; i < blocks-len(buffers); i++ {
		buffers = append(buffers, make([]byte, blockSize))
	}

	bs := buffers[:blocks]
	if size%blockSize != 0 {
		bs[blocks-1] = bs[blocks-1][:size%blockSize]
	}
	return bs
}

func readAndPrint(iour *iouring.IOURing, file *os.File) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()
	buffers := getBuffers(size)

	compCh := make(chan *iouring.Result, 1)
	_, err = iour.SubmitRequest(iouring.Readv(int(file.Fd()), buffers), compCh)
	result := <-compCh
	if err := result.Err(); err != nil {
		return result.Err()
	}

	fmt.Println(file.Name(), ":")
	for _, buffer := range *result.GetRequestBuffers() {
		fmt.Printf("%s", buffer)
	}
	fmt.Println()
	return err
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf("Usage: %s file1 file2 ...\n", os.Args[0])
	}

	iour, err := iouring.New(1)
	if err != nil {
		panic(err)
	}
	defer iour.Close()

	for _, filename := range os.Args[1:] {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("open file error: %v\n", err)
			return
		}
		if err := readAndPrint(iour, file); err != nil {
			fmt.Printf("cat %s error: %v\n", filename, err)
			return
		}
	}
}
