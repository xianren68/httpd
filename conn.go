package httpd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
)

type conn struct {
	svc *Server           // server对象
	rwc net.Conn          // tcp 连接
	lr  *io.LimitedReader // 限制请求头的最大尺寸
	bw  *bufio.Writer     // 缓存写入
	br  *bufio.Reader     // 缓存读取
}

func newConn(svc *Server, rwc net.Conn) *conn {
	lr := &io.LimitedReader{R: rwc, N: 1 << 20}
	return &conn{
		svc: svc,
		rwc: rwc,
		lr:  lr,
		br:  bufio.NewReader(lr),
		bw:  bufio.NewWriter(rwc),
	}
}

func (c *conn) serve() {
	defer func() {
		// 处理错误
		if err := recover(); err != nil {
			log.Printf("http: panic serving: %v", err)
		}
		// 关闭tcp连接
		c.close()
	}()
	// http 长连接可能有多个请求，用for解析
	for {
		resp := setupResponse()
		req, err := c.readRequest()
		if err != nil {
			handleErr(err, c)
		}
		// 用户自定义处理请求函数
		c.svc.Handler.ServeHTTP(resp, req)
	}
}

// readRequest 从连接中读取请求头
func (c *conn) readRequest() (*Request, error) {
	return readRequest(c)
}

// handleErr 错误处理
func handleErr(err error, c *conn) {
	fmt.Println(err)
}

// close 关闭连接
func (c *conn) close() {
	// 关闭tcp连接
	_ = c.rwc.Close()
}

func setupResponse() *response {
	return nil
}
