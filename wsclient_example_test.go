package wsclient

import (
  "net/http"
)

func ExampleRegister() {
  type A struct {
    V1 string
    V2 int
  }

  Register("sample1",func(c Client,data A){
  })

  type B struct {
    V3 []byte
    V4 float32
  }

  Register("sample2",func(c Client,data B){
  })
}

func ExampleGetHandler() {
  http.Handle("/ws",  GetHandler())
  http.ListenAndServe(":8080", nil)
}
