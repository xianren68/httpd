package httpd

import (
	"fmt"
	"io"
	"io/ioutil"
	"testing"
)

type myHandler struct{}

// echo回显服务器，将客户端的报文主体原封不动返回
func (*myHandler) ServeHTTP(w ResponseWriter, r *Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	const prefix = "your message:"
	io.WriteString(w, "HTTP/1.1 200 OK\r\n")
	io.WriteString(w, fmt.Sprintf("Content-Length: %d\r\n", len(buf)+len(prefix)))
	io.WriteString(w, "\r\n")
	io.WriteString(w, prefix)
	w.Write(buf)
}
func TestServer(t *testing.T) {
	s := myHandler{}
	svc := Server{
		Addr:    ":10086",
		Handler: &s,
	}
	err := svc.ListenAndServe()
	if err != nil {
		t.Error(err)
	}

}
