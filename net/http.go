package net

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//HTTPClient HTTP客户端
type HTTPClient struct {
	client *http.Client
}
type HTTPClientRequest struct {
	headers  map[string]string
	client   *http.Client
	method   string
	url      string
	params   string
	encoding string
}

//NewHTTPClientCert 根据pem证书初始化httpClient
func NewHTTPClientCert(certFile string, keyFile string, caFile string) (client *HTTPClient, err error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return
	}
	caData, err := ioutil.ReadFile(caFile)
	if err != nil {
		return
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)
	ssl := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}
	ssl.Rand = rand.Reader
	client = &HTTPClient{}
	client.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: ssl,
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, 0)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   0,
			ResponseHeaderTimeout: 0,
		},
	}
	return
}

//NewHTTPClient 构建HTTP客户端，用于发送GET POST等请求
func NewHTTPClient() (client *HTTPClient) {
	client = &HTTPClient{}
	client.client = &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, 0)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   0,
			ResponseHeaderTimeout: 0,
		},
	}
	return
}

//NewHTTPClientProxy 根据代理服务器地址创建httpClient
func NewHTTPClientProxy(proxy string) (client *HTTPClient) {
	client = &HTTPClient{}
	client.client = &http.Client{
		Transport: &http.Transport{
			Proxy: func(_ *http.Request) (*url.URL, error) {
				return url.Parse(proxy) //根据定义Proxy func(*Request) (*url.URL, error)这里要返回url.URL
			},
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, 0)
				if err != nil {
					return nil, err
				}
				return c, nil
			},
			MaxIdleConnsPerHost:   0,
			ResponseHeaderTimeout: 0,
		},
	}
	return
}

func (c *HTTPClient) NewRequest(method string, url string, args ...string) *HTTPClientRequest {
	request := &HTTPClientRequest{}
	request.client = c.client
	request.headers = make(map[string]string)
	request.method = strings.ToUpper(method)
	request.params = ""
	request.url = url
	request.encoding = getEncoding(args...)
	return request
}
func (c *HTTPClientRequest) SetData(params string) {
	c.params = params
}
func (c *HTTPClientRequest) SetHeader(key string, value string) {
	c.headers[key] = value
}

//Request 发送http请求, method:http请求方法包括:get,post,delete,put等 url: 请求的HTTP地址,不包括参数,params:请求参数,
//header,http请求头多个用/n分隔,每个键值之前用=号连接
func (c *HTTPClientRequest) Request() (content string, status int, err error) {
	req, err := http.NewRequest(c.method, c.url, strings.NewReader(c.params))	
	if err != nil {
		return
	}
	req.Close = true
	for i, v := range c.headers {
		req.Header.Set(i, v)
	}
	resp, err := c.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	status = resp.StatusCode
	content, err = changeEncodingData(c.encoding, body)

	return
}

//Get http get请求
func (c *HTTPClient) Get(url string, args ...string) (content string, status int, err error) {
	encoding := getEncoding(args...)
	resp, err := c.client.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	status = resp.StatusCode
	content, err = changeEncodingData(encoding, body)
	return
}

//Post http Post请求
func (c *HTTPClient) Post(url string, params string, args ...string) (content string, status int, err error) {
	encoding := getEncoding(args...)
	resp, err := c.client.Post(url, "application/x-www-form-urlencoded", strings.NewReader(params))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	status = resp.StatusCode
	content, err = changeEncodingData(encoding, body)
	return
}

func getEncoding(params ...string) (encoding string) {
	if len(params) > 0 {
		encoding = strings.ToUpper(params[0])
		return
	}
	return "UTF-8"
}
func changeEncodingData(encoding string, data []byte) (content string, err error) {
	if !strings.EqualFold(encoding, "GBK") && !strings.EqualFold(encoding, "GB2312") {
		content = string(data)
		return
	}
	buffer, err := ioutil.ReadAll(transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder()))
	if err != nil {
		return
	}
	content = string(buffer)
	return
}
