package httpd

// Header 定义header类型
type Header map[string][]string

// Add 往指定键中添加数据
func (h Header) Add(key, value string) {
	h[key] = append(h[key], value)
}

// Set 往请求头中添加键值对
func (h Header) Set(key, value string) {
	h[key] = []string{value}
}

// Get 获取请求头数据
func (h Header) Get(key string) string {
	if value, ok := h[key]; ok && len(value) > 0 {
		return value[0]
	}
	return ""
}

// Del 删除指定键值对
func (h Header) Del(key string) {
	delete(h, key)
}
