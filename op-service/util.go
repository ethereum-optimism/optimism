package op_service

func PrefixEnvVar(prefix, suffix string) string {
	return prefix + "_" + suffix
}
