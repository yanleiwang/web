package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

type StringValue struct {
	val string
	err error
}

func (s StringValue) String() (string, error) {
	return s.val, s.err
}

func (s StringValue) ToInt64() (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return strconv.ParseInt(s.val, 10, 64)
}

type Context struct {
	Req              *http.Request
	Resp             http.ResponseWriter
	PathParams       map[string]string
	cacheQueryValues url.Values
}

// BindJSON 将输入Body中的Json字符串反序列化到val上
func (c *Context) BindJSON(val any) error {
	if c.Req.Body == nil {
		return errors.New("web: body is nil")
	}
	return json.NewDecoder(c.Req.Body).Decode(val)
}

func (c *Context) QueryValue(key string) StringValue {

	if c.cacheQueryValues == nil {
		c.cacheQueryValues = c.Req.URL.Query()
	}
	vals, ok := c.cacheQueryValues[key]
	if !ok {
		return StringValue{
			val: "",
			err: errors.New("web, 找不到这个key"),
		}
	}

	return StringValue{
		val: vals[0],
		err: nil,
	}

}

func (c *Context) PathValue(key string) StringValue {
	val, ok := c.PathParams[key]
	if !ok {
		return StringValue{
			val: "",
			err: errors.New("web: 找不到这个key"),
		}
	}
	return StringValue{
		val: val,
		err: nil,
	}
}

func (c *Context) RespJSON(code int, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	c.Resp.WriteHeader(code)
	_, err = c.Resp.Write(data)
	return err

}
