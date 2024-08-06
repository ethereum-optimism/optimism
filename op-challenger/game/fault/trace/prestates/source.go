package prestates

import "net/url"

func NewPrestateSource(baseURL *url.URL, path string, localDir string) PrestateSource {
	if baseURL != nil {
		return NewMultiPrestateProvider(baseURL, localDir)
	} else {
		return NewSinglePrestateSource(path)
	}
}
