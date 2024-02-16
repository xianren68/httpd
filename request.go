package httpd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
)

type Request struct {
	Method      string            // 请求方法
	URL         *url.URL          // url
	Proto       string            // 协议及版本
	Header      Header            // 请求头
	Body        io.Reader         // 请求主体
	RemoteAddr  string            // 客户端地址
	RequestURI  string            // 字符串形式的url
	conn        *conn             // 产生此request的http连接
	cookies     map[string]string // cookie
	queryString map[string]string // 存储query参数
}

// readRequest 从tcp连接中读出request.
func readRequest(conn *conn) (r *Request, err error) {
	r = new(Request)
	r.conn = conn
	r.RemoteAddr = conn.rwc.RemoteAddr().String()
	// 读取第一行，获取请求方法，协议，网址
	line, err := readline(conn.br)
	if err != nil {
		return
	}
	// 按空格分割信息
	_, err = fmt.Sscanf(string(line), "%s%s%s", &r.Method, &r.RequestURI, &r.Proto)
	if err != nil {
		return
	}
	// 将URI转为url
	r.URL, err = url.ParseRequestURI(r.RequestURI)
	if err != nil {
		return
	}
	// 解析query参数
	r.parseQuery()
	// 读取header
	r.Header, err = readHeader(conn.br)
	if err != nil {
		return
	}
	// 获取请求体时不再限制大小
	const nolimit = (1 << 63) - 1
	r.conn.lr.N = nolimit
	// 设置请求体
	r.setupBody()

	return r, nil
}

// readline 读取一行数据.
func readline(br *bufio.Reader) ([]byte, error) {
	line, prefix, err := br.ReadLine()
	if err != nil {
		return line, err
	}
	// prefix是为了防止一行数据超过设置的缓存大小还没读完
	var l []byte
	for prefix {
		l, prefix, err = br.ReadLine()
		if err != nil {
			break
		}
		line = append(line, l...)
	}
	return line, err
}

// parseQuery 解析出query参数.
func (r *Request) parseQuery() {
	r.queryString = parseQuery(r.URL.RawQuery)
}

// parseQuery 解析出query参数.
func parseQuery(RawQuery string) map[string]string {
	splits := strings.Split(RawQuery, "&")
	queries := make(map[string]string, len(splits))
	for _, sp := range splits {
		index := strings.IndexByte(sp, '=')
		if index == -1 || index == len(sp)-1 {
			continue
		}
		queries[strings.TrimSpace(sp[:index])] = strings.TrimSpace(sp[index+1:])

	}
	return queries
}

// readHeader 读取请求头.
func readHeader(br *bufio.Reader) (Header, error) {
	header := make(Header)
	for {
		line, err := readline(br)
		if err != nil {
			return nil, err
		}
		// 读到空行表示请求结束
		if len(line) == 0 {
			break
		}
		// 读取每个键值对
		indexByte := bytes.IndexByte(line, ':')
		if indexByte == -1 {
			return nil, errors.New("unsupported protocol")
		}
		if indexByte == len(line)-1 {
			continue
		}
		k, v := string(line[:indexByte]), strings.TrimSpace(string(line[indexByte+1:]))
		header[k] = append(header[k], v)
	}
	return header, nil
}

type eofReader struct{}

// 实现io.Reader接口.
func (er *eofReader) Read([]byte) (n int, err error) {
	return 0, io.EOF
}

// setupBody 设置请求体.
func (r *Request) setupBody() {
	// 只有post和put请求携带结构体
	if r.Method != "POST" && r.Method != "PUT" {
		r.Body = &eofReader{}
	} else if r.chunked() {
		r.Body = &chunkReader{br: r.conn.br}
		r.fixExpectContinueReader()
	} else if cl := r.Header.Get("Content-Length"); cl != "" {
		// 设置读取的最大长度
		contentLength, err := strconv.ParseInt(cl, 10, 64)
		if err != nil {
			r.Body = &eofReader{}
			return
		}
		r.Body = io.LimitReader(r.conn.br, contentLength)
		r.fixExpectContinueReader()

	} else {
		r.Body = &eofReader{}
	}
}

// Cookie 获取指定cookie值.
func (r *Request) Cookie(name string) string {
	// 将cookie解析置后，使用时才解析
	if r.cookies == nil {
		r.parseCookies()
	}
	return r.cookies[name]
}

// Query 获取指定的query参数.
func (r *Request) Query(name string) string {
	return r.queryString[name]
}

// parseCookies 解析cookie.
func (r *Request) parseCookies() {
	if r.cookies != nil {
		return
	}
	r.cookies = make(map[string]string)
	rawCookies, ok := r.Header["Cookie"]
	if !ok {
		return
	}
	for _, line := range rawCookies {
		// example(line): uuid=123456; HOME=1
		kvs := strings.Split(line, ";")
		if len(kvs) == 1 && kvs[0] == "" {
			continue
		}
		for i := 0; i < len(kvs); i++ {
			// example(kvs[i]): uuid=123456
			indexByte := strings.IndexByte(kvs[i], '=')
			if indexByte == -1 {
				continue
			}
			r.cookies[strings.TrimSpace(kvs[i][:indexByte])] = strings.TrimSpace(kvs[i][indexByte+1:])
		}
	}

}

// chunked 判断http协议是否使用chunk.
func (r *Request) chunked() bool {
	te := r.Header.Get("Transfer-Encoding")
	return te == "chunked"
}

// expectContinueReader 用于处理有些请求需要100 continue字段才会继续发送body.
type expectContinueReader struct {
	// 判断是否已经发送过 100 continue
	wroteContinue bool
	r             io.Reader
	w             *bufio.Writer
}

// Read 实现io.Reader.
func (ex *expectContinueReader) Read(p []byte) (n int, err error) {
	// 判断是否发送过 100 continue
	if !ex.wroteContinue {
		_, _ = ex.w.WriteString("HTTP/1.1 100 Continue\r\n\r\n")
		_ = ex.w.Flush()
		ex.wroteContinue = true
	}
	return ex.r.Read(p)
}

// fixExpectContinueReader 处理需要发送 100 continue 来继续接收请求体信息的连接.
func (r *Request) fixExpectContinueReader() {
	if r.Header.Get("Expect") != "100-continue" {
		return
	}
	r.Body = &expectContinueReader{
		r: r.Body,
		w: r.conn.bw,
	}
}

// finishRequest 一次请求后的收尾工作，防止影响后续请求
func (r *Request) finishRequest() (err error) {
	// 将缓冲区信息刷新到tcp连接中
	err = r.conn.bw.Flush()
	if err != nil {
		return
	}
	// 将多余的请求体信息删除
	_, err = io.Copy(io.Discard, r.Body)
	return
}
