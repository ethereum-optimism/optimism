package oppprof

import (
	"errors"
	"fmt"
	"math"
	"strings"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	"github.com/ethereum-optimism/optimism/op-service/flags"
	"github.com/urfave/cli/v2"
)

const (
	EnabledFlagName     = "pprof.enabled"
	ListenAddrFlagName  = "pprof.addr"
	PortFlagName        = "pprof.port"
	ProfileTypeFlagName = "pprof.type"
	ProfilePathFlagName = "pprof.path"
	defaultListenAddr   = "0.0.0.0"
	defaultListenPort   = 6060
)

var ErrInvalidPort = errors.New("invalid pprof port")
var allowedProfileTypes = []profileType{"cpu", "heap", "goroutine", "threadcreate", "block", "mutex", "allocs"}

type profileType string

func (t profileType) String() string {
	return string(t)
}

func (t *profileType) Set(value string) error {
	if !validProfileType(profileType(value)) {
		return fmt.Errorf("unknown profile type: %q", value)
	}
	*t = profileType(value)
	return nil
}

func (t *profileType) Clone() any {
	cpy := *t
	return &cpy
}

func validProfileType(value profileType) bool {
	for _, k := range allowedProfileTypes {
		if k == value {
			return true
		}
	}
	return false
}

func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		ListenEnabled: false,
		ListenAddr:    defaultListenAddr,
		ListenPort:    defaultListenPort,
	}
}

func CLIFlags(envPrefix string) []cli.Flag {
	return CLIFlagsWithCategory(envPrefix, "")
}

func CLIFlagsWithCategory(envPrefix string, category string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:     EnabledFlagName,
			Usage:    "Enable the pprof server",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PPROF_ENABLED"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     ListenAddrFlagName,
			Usage:    "pprof listening address",
			Value:    defaultListenAddr, // TODO: Switch to 127.0.0.1
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PPROF_ADDR"),
			Category: category,
		},
		&cli.IntFlag{
			Name:     PortFlagName,
			Usage:    "pprof listening port",
			Value:    defaultListenPort,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PPROF_PORT"),
			Category: category,
		},
		&cli.GenericFlag{
			Name:     ProfilePathFlagName,
			Usage:    "pprof file path. If it is a directory, the path is {dir}/{profileType}.prof",
			Value:    new(flags.PathFlag),
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PPROF_PATH"),
			Category: category,
		},
		&cli.GenericFlag{
			Name:  ProfileTypeFlagName,
			Usage: "pprof profile type. One of " + openum.EnumString(allowedProfileTypes),
			Value: func() *profileType {
				defaultProfType := profileType("")
				return &defaultProfType
			}(),
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PPROF_TYPE"),
			Category: category,
		},
	}
}

type CLIConfig struct {
	ListenEnabled bool
	ListenAddr    string
	ListenPort    int

	ProfileType     profileType
	ProfileDir      string
	ProfileFilename string
}

func (m CLIConfig) Check() error {
	if !m.ListenEnabled {
		return nil
	}

	if m.ListenPort < 0 || m.ListenPort > math.MaxUint16 {
		return ErrInvalidPort
	}

	return nil
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	profilePathFlag := ctx.Generic(ProfilePathFlagName).(*flags.PathFlag)
	return CLIConfig{
		ListenEnabled:   ctx.Bool(EnabledFlagName),
		ListenAddr:      ctx.String(ListenAddrFlagName),
		ListenPort:      ctx.Int(PortFlagName),
		ProfileType:     profileType(strings.ToLower(ctx.String(ProfileTypeFlagName))),
		ProfileDir:      profilePathFlag.Dir(),
		ProfileFilename: profilePathFlag.Filename(),
	}
}
