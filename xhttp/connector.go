package xhttp

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type MethodType string

const (
	GET    MethodType = "GET"
	POST   MethodType = "POST"
	DELETE MethodType = "DELETE"
)

const (
	plainText     string = "text/plain"
	htmlText      string = "text/html"
	formUrlEncode string = "application/x-www-form-urlencoded"
	json          string = "application/json"
)

type Connector struct {
	uri    string
	method MethodType
}

func NewConnector(method MethodType, uri string) *Connector {
	conn := new(Connector)
	conn.method = method
	conn.uri = uri

	return conn
}

func createClient(uri string) *http.Client {
	var (
		client *http.Client
	)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if strings.Contains(uri, "https://") {
		client = &http.Client{Transport: tr}
	} else {
		client = &http.Client{}
	}

	return client
}

func (c *Connector) httpRequest(contentType, data string, additionalHeader map[string]string) (string, error) {
	var body io.Reader
	if c.method == GET {
		body = nil
	} else {
		body = strings.NewReader(data)
	}
	request, err := http.NewRequest(string(c.method), c.uri, body)
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", contentType)
	if additionalHeader != nil {
		for k, v := range additionalHeader {
			request.Header.Add(k, v)
		}
	}

	client := createClient(c.uri)
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	return string(result), nil
}

func (c *Connector) TEXT(text string, additionalHeader map[string]string) (string, error) {
	return c.httpRequest(plainText, text, additionalHeader)
}

func (c *Connector) HTML(html string, additionalHeader map[string]string) (string, error) {
	return c.httpRequest(htmlText, html, additionalHeader)
}

func (c *Connector) FORM(keyValue, additionalHeader map[string]string) (string, error) {
	path := url.Values{}
	for k, v := range keyValue {
		path.Add(k, v)
	}
	data := path.Encode()

	return c.httpRequest(formUrlEncode, data, additionalHeader)
}

func (c *Connector) JSON(jsonString string, additionalHeader map[string]string) (string, error) {
	return c.httpRequest(json, jsonString, additionalHeader)
}

func (c *Connector) MULTIPART(filePath string, keyValue, additionalHeader map[string]string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	fi, _ := file.Stat()
	_ = file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	if keyValue != nil {
		for k, v := range keyValue {
			_ = writer.WriteField(k, v)
		}
	}

	filePart, err := writer.CreateFormFile("file", fi.Name())
	if err != nil {
		return "", err
	}
	_, err = filePart.Write(fileContents)
	if err != nil {
		return "", err
	}
	_ = writer.Close()

	request, err := http.NewRequest(string(c.method), c.uri, body)
	if err != nil {
		return "", err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())
	if additionalHeader != nil {
		for k, v := range additionalHeader {
			request.Header.Add(k, v)
		}
	}

	client := createClient(c.uri)
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	return string(result), nil
}
