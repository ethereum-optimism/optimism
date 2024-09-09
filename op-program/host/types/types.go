package types

type DataFormat string

const (
	DataFormatFile   DataFormat = "file"
	DataFormatPebble DataFormat = "pebble"
)

var SupportedDataFormats = []DataFormat{DataFormatFile, DataFormatPebble}
