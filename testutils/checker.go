package testutils

type Request struct {
	Path   string
	Body   string
	Method string
}

type Response struct {
	Status  int
	Headers map[string]string
	Body    string
}

type Mock struct {
	Request
	Response
}
