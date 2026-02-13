package endpoint

// RestHTTP is an interface for an endpoint to provide flexibility.
// By default the RestHTTP just returns an REST-ful HTTP endpoint string.
// But the RestHTTP can implement one or more extension interfaces,
// to provide alternative ways of establishing a connection,
// or even a fully initialized client binding.
type RestHTTP interface {
	RestHTTP() string
}

// RestHTTPURL is an HTTP endpoint URL string
type RestHTTPURL string

func (url RestHTTPURL) RestHTTP() string {
	return string(url)
}
