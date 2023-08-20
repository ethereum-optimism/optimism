package version

var (
	Version = "v0.1.0"
	Meta    = "dev"
)

var SimpleWithMeta = func() string {
	v := Version
	if Meta != "" {
		v += "-" + Meta
	}
	return v
}()
