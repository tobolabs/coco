package coco

const (
	mediaTypeRegex = `^([a-z]+\/[a-z0-9\-\.]+)(?:\;(.+))?$`
)

type negotiator struct {
	req *Request
}
