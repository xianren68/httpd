package httpd

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

type chunkReader struct {
	// 当前块还有多少字节未读
	n  int
	br *bufio.Reader
	// done标志主体是否读完
	done bool
	crlf [2]byte // 读取\r\n
}

// Read 读取chunk块.
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	// 判断报文是否读完
	if cr.done {
		return 0, io.EOF
	}
	// 当前块没有数据，准备读取下一块
	if cr.n == 0 {
		// 先读取chunk size
		cr.n, err = cr.getChunkSize()
		if err != nil {
			return
		}

	}
	// 块中无数据，读取到了末尾
	if cr.n == 0 {
		// 修改标志位
		cr.done = true
		err = cr.discardCRLF()
		return
	}
	// 剩余的数据大于要读取的长度
	if len(p) < cr.n {
		n, err = cr.br.Read(p)
		cr.n -= n
		return
	}
	// 剩余的数据小于等于要读取的长度
	n, _ = io.ReadFull(cr.br, p[:cr.n])
	cr.n = 0
	// 将\r\n消费
	if err = cr.discardCRLF(); err != nil {
		return
	}
	return
}

// getChunkSize 获取每个chunk的字节数.
func (cr *chunkReader) getChunkSize() (int, error) {
	line, err := readline(cr.br)
	if err != nil {
		return 0, err
	}
	// chunk size为16进制数据，将其转为10进制
	n, err := strconv.ParseInt(string(line), 16, 64)
	if err != nil {
		return int(n), err
	}
	return int(n), err
}

// discardCRLF 消费掉末尾的CRLF
func (cr *chunkReader) discardCRLF() (err error) {
	_, err = io.ReadFull(cr.br, cr.crlf[:])
	if err != nil {
		return err
	}
	if cr.crlf[0] != '\r' || cr.crlf[1] != '\n' {
		return errors.New("unsupported encoding format of chunk")
	}
	return
}
