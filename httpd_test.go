package httpd

import (
	"fmt"
	"testing"
)

type server struct {
}

func (s *server) ServeHTTP(w ResponseWriter, r *Request) {
	fmt.Println("hello world!")
}
func TestServer(t *testing.T) {
	s := server{}
	svc := Server{
		Addr:    ":8080",
		Handler: &s,
	}
	err := svc.ListenAndServe()
	if err != nil {
		t.Error(err)
	}

}
