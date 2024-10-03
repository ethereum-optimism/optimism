package state

import "net/url"

type ArtifactsURL url.URL

func (a *ArtifactsURL) MarshalText() ([]byte, error) {
	return []byte((*url.URL)(a).String()), nil
}

func (a *ArtifactsURL) UnmarshalText(text []byte) error {
	u, err := url.Parse(string(text))
	if err != nil {
		return err
	}
	*a = ArtifactsURL(*u)
	return nil
}

func ParseArtifactsURL(in string) (*ArtifactsURL, error) {
	u, err := url.Parse(in)
	if err != nil {
		return nil, err
	}
	return (*ArtifactsURL)(u), nil
}

func MustParseArtifactsURL(in string) *ArtifactsURL {
	u, err := ParseArtifactsURL(in)
	if err != nil {
		panic(err)
	}
	return u
}
