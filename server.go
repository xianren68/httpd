package httpd

import "net"

type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}

type Server struct {
	Addr    string
	Handler Handler
}

func (svc *Server) ListenAndServe() error {
	// 创建tcp连接
	l, err := net.Listen("tcp", svc.Addr)
	if err != nil {
		return err
	}
	for {
		// 不断的监听新的连接
		accept, err := l.Accept()
		if err != nil {
			continue
		}
		// 开启一个新协程处理连接
		conn := newConn(svc, accept)
		go conn.serve()
	}
}
