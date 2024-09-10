package types

type DataFormat string

const (
	DataFormatFile      DataFormat = "file"
	DataFormatDirectory DataFormat = "directory"
	DataFormatPebble    DataFormat = "pebble"
)

var SupportedDataFormats = []DataFormat{DataFormatFile, DataFormatDirectory, DataFormatPebble}
