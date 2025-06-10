package xhttp

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
