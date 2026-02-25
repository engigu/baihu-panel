package main

import (
	"fmt"
	"os"

	yeka_zip "github.com/yeka/zip"
)

func main() {
	f, err := os.Create("test.zip")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	zw := yeka_zip.NewWriter(f)
	defer zw.Close()

	w, err := zw.Encrypt("encrypted.txt", "123", yeka_zip.AES256Encryption)
	if err != nil {
		panic(err)
	}
	w.Write([]byte("encrypted data"))

	w2, err := zw.Create("plain.txt")
	if err != nil {
		panic(err)
	}
	w2.Write([]byte("plain data"))
	fmt.Println("success")
}
