package main

type JsonOutput struct {
	AST AST        `json:"ast"`
	ABI []ABIEntry `json:"abi"`
}
type ABIEntry struct {
	Type            string      `json:"type"`
	Name            string      `json:"name"`
	Inputs          []ABIInput  `json:"inputs"`
	Outputs         []ABIOutput `json:"outputs"`
	StateMutability string      `json:"stateMutability"`
}

type ABIInput struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	InternalType string `json:"internalType"`
}

type ABIOutput struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	InternalType string `json:"internalType"`
}

type AST struct {
	AbsolutePath    string           `json:"absolutePath"`
	ID              int              `json:"id"`
	ExportedSymbols map[string][]int `json:"exportedSymbols"`
	NodeType        string           `json:"nodeType"`
	Src             string           `json:"src"`
	Nodes           []Node           `json:"nodes"`
	License         string           `json:"license"`
}

type Node struct {
	ID                      int              `json:"id"`
	NodeType                string           `json:"nodeType"`
	Src                     string           `json:"src"`
	Nodes                   []Node           `json:"nodes,omitempty"`
	Literals                []string         `json:"literals,omitempty"`
	AbsolutePath            string           `json:"absolutePath,omitempty"`
	File                    string           `json:"file,omitempty"`
	NameLocation            string           `json:"nameLocation,omitempty"`
	Scope                   int              `json:"scope,omitempty"`
	SourceUnit              int              `json:"sourceUnit,omitempty"`
	SymbolAliases           []SymbolAlias    `json:"symbolAliases,omitempty"`
	UnitAlias               string           `json:"unitAlias,omitempty"`
	CanonicalName           string           `json:"canonicalName,omitempty"`
	Name                    string           `json:"name,omitempty"`
	UnderlyingType          *TypeName        `json:"underlyingType,omitempty"`
	Members                 []Member         `json:"members,omitempty"`
	Visibility              string           `json:"visibility,omitempty"`
	Constant                bool             `json:"constant,omitempty"`
	Mutability              string           `json:"mutability,omitempty"`
	StateVariable           bool             `json:"stateVariable,omitempty"`
	StorageLocation         string           `json:"storageLocation,omitempty"`
	TypeDescriptions        TypeDescriptions `json:"typeDescriptions,omitempty"`
	TypeName                *TypeName        `json:"typeName,omitempty"`
	Body                    *Body            `json:"body,omitempty"`
	Implemented             bool             `json:"implemented,omitempty"`
	Kind                    string           `json:"kind,omitempty"`
	Modifiers               []interface{}    `json:"modifiers,omitempty"`
	Parameters              *ParameterList   `json:"parameters,omitempty"`
	ReturnParameters        *ParameterList   `json:"returnParameters,omitempty"`
	StateMutability         string           `json:"stateMutability,omitempty"`
	Virtual                 bool             `json:"virtual,omitempty"`
	Abstract                bool             `json:"abstract,omitempty"`
	BaseContracts           []BaseContract   `json:"baseContracts,omitempty"`
	ContractDependencies    []int            `json:"contractDependencies,omitempty"`
	ContractKind            string           `json:"contractKind,omitempty"`
	FullyImplemented        bool             `json:"fullyImplemented,omitempty"`
	LinearizedBaseContracts []int            `json:"linearizedBaseContracts,omitempty"`
	UsedErrors              []int            `json:"usedErrors,omitempty"`
	UsedEvents              []int            `json:"usedEvents,omitempty"`
	FunctionSelector        string           `json:"functionSelector,omitempty"`
	EventSelector           string           `json:"eventSelector,omitempty"`
	Indexed                 bool             `json:"indexed,omitempty"`
	ErrorSelector           string           `json:"errorSelector,omitempty"`
}

type SymbolAlias struct {
	Foreign      *Identifier `json:"foreign"`
	NameLocation string      `json:"nameLocation"`
	Local        string      `json:"local"`
}

type Identifier struct {
	ID                     int              `json:"id"`
	Name                   string           `json:"name"`
	NodeType               string           `json:"nodeType"`
	OverloadedDeclarations []int            `json:"overloadedDeclarations"`
	ReferencedDeclaration  int              `json:"referencedDeclaration"`
	Src                    string           `json:"src"`
	TypeDescriptions       TypeDescriptions `json:"typeDescriptions"`
}

type TypeName struct {
	ID               int              `json:"id"`
	Name             string           `json:"name"`
	NodeType         string           `json:"nodeType"`
	Src              string           `json:"src"`
	TypeDescriptions TypeDescriptions `json:"typeDescriptions"`
	BaseType         *TypeName        `json:"baseType"`
	KeyType          *TypeName        `json:"keyType"`
	ValueType        *TypeName        `json:"valueType"`
	KeyName          string           `json:"keyName"`
	ValueName        string           `json:"valueName"`
}

type TypeDescriptions struct {
	TypeIdentifier string `json:"typeIdentifier"`
	TypeString     string `json:"typeString"`
}

type Member struct {
	Constant         bool             `json:"constant"`
	ID               int              `json:"id"`
	Mutability       string           `json:"mutability"`
	Name             string           `json:"name"`
	NameLocation     string           `json:"nameLocation"`
	NodeType         string           `json:"nodeType"`
	Scope            int              `json:"scope"`
	Src              string           `json:"src"`
	StateVariable    bool             `json:"stateVariable"`
	StorageLocation  string           `json:"storageLocation"`
	TypeDescriptions TypeDescriptions `json:"typeDescriptions"`
	TypeName         *TypeName        `json:"typeName"`
	Visibility       string           `json:"visibility"`
}

type Body struct {
	ID         int    `json:"id"`
	NodeType   string `json:"nodeType"`
	Src        string `json:"src"`
	Statements []Node `json:"statements"`
}

type ParameterList struct {
	ID         int    `json:"id"`
	NodeType   string `json:"nodeType"`
	Parameters []Node `json:"parameters"`
	Src        string `json:"src"`
}

type BaseContract struct {
	BaseName *Identifier `json:"baseName"`
	ID       int         `json:"id"`
	NodeType string      `json:"nodeType"`
	Src      string      `json:"src"`
}
