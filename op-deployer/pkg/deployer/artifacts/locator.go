package artifacts

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

type schemeUnmarshaler func(string) (*Locator, error)

var schemeUnmarshalerDispatch = map[string]schemeUnmarshaler{
	"file":  unmarshalURL,
	"http":  unmarshalURL,
	"https": unmarshalURL,
}

const EmbeddedLocatorString = "embedded"

var embeddedURL = &url.URL{
	Scheme: EmbeddedLocatorString,
}

var EmbeddedLocator = &Locator{
	URL: embeddedURL,
}

var DefaultL1ContractsLocator = EmbeddedLocator

var DefaultL2ContractsLocator = EmbeddedLocator

func NewLocatorFromURL(u string) (*Locator, error) {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	return &Locator{
		URL: parsedURL,
	}, nil
}

func MustNewLocatorFromURL(u string) *Locator {
	loc, err := NewLocatorFromURL(u)
	if err != nil {
		panic(err)
	}
	return loc
}

func MustNewFileLocator(path string) *Locator {
	loc, err := NewFileLocator(path)
	if err != nil {
		panic(err)
	}
	return loc
}

type Locator struct {
	URL *url.URL
}

func NewFileLocator(path string) (*Locator, error) {
	u, err := url.Parse("file://" + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	return &Locator{URL: u}, nil
}

func (a *Locator) UnmarshalText(text []byte) error {
	str := string(text)

	if strings.HasPrefix(str, "tag://") {
		return errors.New("tag:// locators are no longer supported - use embedded artifacts instead")
	}

	if str == "embedded" {
		*a = *EmbeddedLocator
		return nil
	}

	for scheme, unmarshaler := range schemeUnmarshalerDispatch {
		if !strings.HasPrefix(str, scheme+":") {
			continue
		}

		loc, err := unmarshaler(str)
		if err != nil {
			return err
		}

		*a = *loc
		return nil
	}

	return fmt.Errorf("unsupported scheme %s", str)
}

func (a *Locator) MarshalText() ([]byte, error) {
	if a.URL.String() == embeddedURL.String() {
		return []byte("embedded"), nil
	}

	return []byte(a.URL.String()), nil
}

func (a *Locator) Equal(b *Locator) bool {
	aStr, _ := a.MarshalText()
	bStr, _ := b.MarshalText()
	return string(aStr) == string(bStr)
}

func (a *Locator) IsEmbedded() bool {
	return a.URL.String() == embeddedURL.String()
}

func unmarshalURL(text string) (*Locator, error) {
	u, err := url.Parse(text)
	if err != nil {
		return nil, err
	}

	return &Locator{URL: u}, nil
}
