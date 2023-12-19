package hardhat

import "strings"

type QualifiedName struct {
	SourceName   string
	ContractName string
}

func ParseFullyQualifiedName(name string) QualifiedName {
	names := strings.Split(name, ":")
	if len(names) == 1 {
		return QualifiedName{
			SourceName:   "",
			ContractName: names[0],
		}
	}

	contractName := names[len(names)-1]
	sourceName := strings.Join(names[0:len(names)-1], ":")

	return QualifiedName{
		ContractName: contractName,
		SourceName:   sourceName,
	}
}

func GetFullyQualifiedName(sourceName, contractName string) string {
	return sourceName + ":" + contractName
}

func IsFullyQualifiedName(name string) bool {
	return strings.Contains(name, ":")
}
