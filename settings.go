package coco

type settings struct {
	env string

	xPoweredBy string

	eTag bool

	viewCache bool

	trustProxy bool

	subDomainOffset int

	strictRouting bool
}

func (s *settings) setInt(key string, val int) {
	switch key {
	case "subdomain offset":
		s.subDomainOffset = val
	}
}

func (s *settings) setBool(key string, val bool) {
	switch key {
	case "etag":
		s.eTag = val
	case "view cache":
		s.viewCache = val
	case "trust proxy":
		s.trustProxy = val
	case "strict routing":
		s.strictRouting = val

	}
}

func (s *settings) setString(key, val string) {

	switch key {
	case "env":
		s.env = val
	case "x-powered-by":
		s.xPoweredBy = val
	}
}

func (s *settings) getBool(key string) bool {
	switch key {
	case "etag":
		return s.eTag
	case "view cache":
		return s.viewCache
	case "trust proxy":
		return s.trustProxy
	case "strict routing":
		return s.strictRouting

	}
	return false
}

func (s *settings) SetX(key, val interface{}) {

	switch val.(type) {
	case string:
		s.setString(key.(string), val.(string))
	case bool:
		s.setBool(key.(string), val.(bool))
	case int:
		s.setInt(key.(string), val.(int))
	}
}

func (s *settings) Disable(key string) {
	s.setBool(key, false)
}

func (s *settings) Enable(key string) {
	s.setBool(key, true)
}

func (s *settings) GetX(key string) interface{} {

	switch key {
	case "env":
		return s.env
	case "x-powered-by":
		return s.xPoweredBy
	case "etag":
		return s.eTag
	case "view cache":
		return s.viewCache
	case "trust proxy":
		return s.trustProxy
	case "strict routing":
		return s.strictRouting
	case "subdomain offset":
		return s.subDomainOffset
	}
	return nil
}

func (s *settings) Disabled(key string) bool {
	return !s.getBool(key)
}

func (s *settings) Enabled(key string) bool {
	return s.getBool(key)
}
