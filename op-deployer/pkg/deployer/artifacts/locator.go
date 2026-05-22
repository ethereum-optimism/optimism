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

func CreateHttpLocator(contentHash string) string {
	return fmt.Sprintf("https://storage.googleapis.com/oplabs-contract-artifacts/artifacts-v1-%s.tar.gz", contentHash)
}

var DefaultL1ContractsLocator = &Locator{
	URL: &url.URL{},
}

var DefaultL2ContractsLocator = &Locator{
	URL: &url.URL{},
}

func NewLocatorFromURL(u string) (*Locator, error) {
	if u == "" {
		return nil, fmt.Errorf("artifacts locator cannot be empty")
	}
	if u == "embedded" {
		return nil, errors.New("embedded artifacts are no longer supported - use a file://, http://, or https:// artifacts locator")
	}

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
	if str == "" {
		*a = Locator{URL: &url.URL{}}
		return nil
	}

	if strings.HasPrefix(str, "tag://") {
		return errors.New("tag:// locators are no longer supported - use a file://, http://, or https:// artifacts locator")
	}

	if str == "embedded" {
		return errors.New("embedded artifacts are no longer supported - use a file://, http://, or https:// artifacts locator")
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
	if a == nil || a.URL == nil || a.URL.String() == "" {
		return []byte(""), nil
	}

	return []byte(a.URL.String()), nil
}

func (a *Locator) MarshalTOML() ([]byte, error) {
	if a == nil || a.URL == nil || a.URL.String() == "" {
		return []byte(`""`), nil
	}
	return []byte(`"` + a.URL.String() + `"`), nil
}

func (a *Locator) UnmarshalTOML(i interface{}) error {
	switch v := i.(type) {
	case string:
		return a.UnmarshalText([]byte(v))
	case []byte:
		return a.UnmarshalText(v)
	default:
		return fmt.Errorf("unsupported type for TOML unmarshaling: %T", i)
	}
}

func (a *Locator) Equal(b *Locator) bool {
	aStr, _ := a.MarshalText()
	bStr, _ := b.MarshalText()
	return string(aStr) == string(bStr)
}

func (a *Locator) IsZero() bool {
	return a == nil || a.URL == nil || a.URL.String() == ""
}

func unmarshalURL(text string) (*Locator, error) {
	u, err := url.Parse(text)
	if err != nil {
		return nil, err
	}

	return &Locator{URL: u}, nil
}
