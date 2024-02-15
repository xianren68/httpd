package httpd

type response struct {
	c *conn
}
type ResponseWriter interface {
	Write([]byte) (n int, err error)
}

func (r *response) Write(p []byte) (n int, err error) {
	return r.c.bw.Write(p)
}
func setupResponse(c *conn) *response {
	return &response{c: c}
}
