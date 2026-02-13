package op_service

func FormatVersion(version string, gitCommit string, gitDate string, meta string) string {
	v := version
	if gitCommit != "" {
		if len(gitCommit) >= 8 {
			v += "-" + gitCommit[:8]
		} else {
			v += "-" + gitCommit
		}
	}
	if gitDate != "" {
		v += "-" + gitDate
	}
	if meta != "" {
		v += "-" + meta
	}
	return v
}
