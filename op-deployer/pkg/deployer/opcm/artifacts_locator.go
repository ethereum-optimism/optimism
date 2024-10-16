package opcm

import (
	"fmt"
	"net/url"
	"strings"
)

type schemeUnmarshaler func(string) (*ArtifactsLocator, error)

var schemeUnmarshalerDispatch = map[string]schemeUnmarshaler{
	"tag":   unmarshalTag,
	"file":  unmarshalURL,
	"https": unmarshalURL,
}

type ArtifactsLocator struct {
	URL *url.URL
	Tag string
}

func (a *ArtifactsLocator) UnmarshalText(text []byte) error {
	str := string(text)

	for scheme, unmarshaler := range schemeUnmarshalerDispatch {
		if !strings.HasPrefix(str, scheme+"://") {
			continue
		}

		loc, err := unmarshaler(str)
		if err != nil {
			return err
		}

		*a = *loc
		return nil
	}

	return fmt.Errorf("unsupported scheme")
}

func (a *ArtifactsLocator) MarshalText() ([]byte, error) {
	if a.URL != nil {
		return []byte(a.URL.String()), nil
	}

	if a.Tag != "" {
		return []byte("tag://" + a.Tag), nil
	}

	return nil, fmt.Errorf("no URL, path or tag set")
}

func (a *ArtifactsLocator) IsTag() bool {
	return a.Tag != ""
}

func unmarshalTag(tag string) (*ArtifactsLocator, error) {
	tag = strings.TrimPrefix(tag, "tag://")
	if !strings.HasPrefix(tag, "op-contracts/") {
		return nil, fmt.Errorf("invalid tag: %s", tag)
	}

	if _, err := StandardArtifactsURLForTag(tag); err != nil {
		return nil, err
	}

	return &ArtifactsLocator{Tag: tag}, nil
}

func unmarshalURL(text string) (*ArtifactsLocator, error) {
	u, err := url.Parse(text)
	if err != nil {
		return nil, err
	}

	return &ArtifactsLocator{URL: u}, nil
}
