package log

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

func TestDynamicLogHandler_SetLogLevel(t *testing.T) {
	var records []*log.Record
	h := log.FuncHandler(func(r *log.Record) error {
		records = append(records, r)
		return nil
	})
	d := NewDynamicLogHandler(log.LvlInfo, h)
	logger := log.New()
	logger.SetHandler(d)
	logger.Info("hello world") // y
	logger.Error("error!")     // y
	logger.Debug("debugging")  // n

	// increase log level
	logger.GetHandler().(LvlSetter).SetLogLevel(log.LvlDebug)

	logger.Info("hello again")        // y
	logger.Debug("can see debug now") // y
	logger.Trace("but no trace")      // n

	// and decrease log level
	logger.GetHandler().(LvlSetter).SetLogLevel(log.LvlWarn)
	logger.Warn("visible warning")           // y
	logger.Info("info should be hidden now") // n
	logger.Error("another error")            // y

	require.Len(t, records, 2+2+2)
	require.Equal(t, records[0].Msg, "hello world")
	require.Equal(t, records[1].Msg, "error!")
	require.Equal(t, records[2].Msg, "hello again")
	require.Equal(t, records[3].Msg, "can see debug now")
	require.Equal(t, records[4].Msg, "visible warning")
	require.Equal(t, records[5].Msg, "another error")
}
