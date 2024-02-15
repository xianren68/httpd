package httpd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
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
	// 设置请求体
	r.setupBody()
	return r, nil
}

// readline 读取一行数据
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

// parseQuery 解析出query参数
func (r *Request) parseQuery() {
	r.queryString = parseQuery(r.URL.RawQuery)
}

// parseQuery 解析出query参数
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

// readHeader 读取请求头
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

// 实现io.Reader接口
func (er *eofReader) Read([]byte) (n int, err error) {
	return 0, io.EOF
}

// setupBody 设置请求体
func (r *Request) setupBody() {
	r.Body = new(eofReader)
}

// Cookie 获取指定cookie值
func (r *Request) Cookie(name string) string {
	// 将cookie解析置后，使用时才解析
	if r.cookies == nil {
		r.parseCookies()
	}
	return r.cookies[name]
}

// Query 获取指定的query参数
func (r *Request) Query(name string) string {
	return r.queryString[name]
}

// parseCookies 解析cookie
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
