package helpers

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func SetupLogger(level string) {
	var l zerolog.Level
	switch level {
	case "debug":
		l = zerolog.DebugLevel
	case "info":
		l = zerolog.InfoLevel
	case "warn":
		l = zerolog.WarnLevel
	case "error":
		l = zerolog.ErrorLevel
	default:
		l = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(l)
}

// stderrColor reports whether the log stream (stderr) should be colored. It
// honors NO_COLOR and falls back to plain output when stderr is not a terminal,
// which keeps piped logs and CI output clean.
func stderrColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())
}

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	color := stderrColor()

	timeFormat := "15:04:05"
	if color {
		timeFormat = "[90m15:04:05[0m"
	}

	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: timeFormat,
		NoColor:    !color,
		PartsOrder: []string{
			zerolog.TimestampFieldName,
			zerolog.LevelFieldName,
			zerolog.MessageFieldName,
		},
		FormatLevel: func(i interface{}) string {
			s, _ := i.(string)
			label, code := levelLabel(s)
			if !color || code == "" {
				return label
			}
			return "[" + code + "m" + label + "[0m"
		},
		FormatMessage: func(i interface{}) string {
			if i == nil {
				return ""
			}
			return "  " + i.(string)
		},
	}
	Log = zerolog.New(output).With().Timestamp().Logger()
}

// levelLabel maps a zerolog level string to its padded label and the SGR color
// code used when color is enabled.
func levelLabel(level string) (label, code string) {
	switch level {
	case "debug":
		return "DEBUG", "36"
	case "info":
		return "INFO ", "32"
	case "warn":
		return "WARN ", "33"
	case "error":
		return "ERROR", "31"
	case "fatal":
		return "FATAL", "31"
	default:
		return "????", ""
	}
}
