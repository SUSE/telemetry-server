package app

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	LOG_LEVEL_DEFAULT slog.Level = slog.LevelInfo
	LOG_STYLE_DEFAULT string     = "TEXT"
	LOG_PATH_DEFAULT  string     = "stderr"
)

type LogLevels struct {
	validLevels map[string]slog.Level
}

func NewLogLevels() (logLevels *LogLevels) {
	logLevels = &LogLevels{
		validLevels: make(map[string]slog.Level),
	}

	// define standard names for log levels
	for _, level := range []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	} {
		logLevels.validLevels[level.String()] = level
	}

	// define alternate names for log levels
	for name, level := range map[string]slog.Level{
		"dbg":     slog.LevelDebug,
		"inf":     slog.LevelInfo,
		"wrn":     slog.LevelWarn,
		"warning": slog.LevelWarn,
		"err":     slog.LevelError,
	} {
		logLevels.validLevels[name] = level
	}

	return
}

func (l *LogLevels) Levels() (levels []string) {
	for level := range l.validLevels {
		levels = append(levels, level)
	}
	return levels
}

func (ls *LogLevels) String() string {
	return "[" + strings.Join(ls.Levels(), ", ") + "]"
}

func (l *LogLevels) Canonicalize(levelName string) string {
	return strings.ToUpper(levelName)
}

func (l *LogLevels) Valid(levelName string) (valid bool) {
	_, valid = l.validLevels[l.Canonicalize(levelName)]
	return
}

func (l *LogLevels) GetLevel(levelName string) (level slog.Level, valid bool) {
	level, valid = l.validLevels[l.Canonicalize(levelName)]
	return
}

type LogStyles struct {
	validStyles map[string]bool
}

func NewLogStyles() (logStyles *LogStyles) {
	logStyles = &LogStyles{
		validStyles: make(map[string]bool),
	}

	for _, style := range []string{
		"TEXT",
		"JSON",
		"SYSLOG",
	} {
		logStyles.validStyles[style] = true
	}

	return
}

func (ls *LogStyles) Styles() (styles []string) {
	for style := range ls.validStyles {
		styles = append(styles, style)
	}
	return styles
}

func (ls *LogStyles) String() string {
	return "[" + strings.Join(ls.Styles(), ", ") + "]"
}

func (ls *LogStyles) Canonicalize(styleName string) string {
	return strings.ToUpper(styleName)
}

func (ls *LogStyles) Valid(styleName string) (valid bool) {
	valid = ls.validStyles[ls.Canonicalize(styleName)]
	return
}

func (ls *LogStyles) GetStyle(styleName string) (style string, valid bool) {
	style = ls.Canonicalize(styleName)
	_, valid = ls.validStyles[style]
	return
}

type LogManager struct {
	levels     *LogLevels
	styles     *LogStyles
	logger     *slog.Logger
	logLevel   *slog.LevelVar
	logStyle   string
	logPath    string
	logFile    *os.File
	logHandler slog.Handler
}

func NewLogManager() (lm *LogManager) {
	lm = new(LogManager)

	// init sub structures
	lm.levels = NewLogLevels()
	lm.styles = NewLogStyles()
	lm.logLevel = new(slog.LevelVar)

	// setup defaults
	lm.logStyle = LOG_STYLE_DEFAULT
	lm.logLevel.Set(LOG_LEVEL_DEFAULT)
	lm.logPath = LOG_PATH_DEFAULT

	return
}

func (lm *LogManager) Logger() *slog.Logger {
	return lm.logger
}

func (lm *LogManager) SetLevel(levelName string) (err error) {
	// if no level is specified default to info
	if levelName == "" {
		levelName = LOG_LEVEL_DEFAULT.String()
	}

	level, valid := lm.levels.GetLevel(levelName)
	if !valid {
		return fmt.Errorf(
			"invalid log level name '%s', must be one of %s (case insensitive)",
			levelName,
			lm.levels,
		)
	}

	// set the log level
	lm.logLevel.Set(level)

	return
}

func (lm *LogManager) SetStyle(styleName string) (err error) {
	// if no style is specified default to text
	if styleName == "" {
		styleName = LOG_STYLE_DEFAULT
	}

	style, valid := lm.styles.GetStyle(styleName)
	if !valid {
		return fmt.Errorf(
			"invalid log style name '%s', must be one of %s (case insensitive)",
			styleName,
			lm.styles,
		)
	}

	lm.logStyle = style

	return
}

func checkLogPathValid(path string) (err error) {
	// path must be non-empty
	if len(path) == 0 {
		return fmt.Errorf("invalid path '%s': must be non-empty", path)
	}

	// must be able to generate an absolute path from provided path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return
	}

	// parent directory must exist, be accessible, and be a directory
	parent := filepath.Dir(absPath)
	dirInfo, err := os.Stat(parent)
	if err != nil {
		return
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("parent '%s' of path '%s' is not a directory", parent, path)
	}

	// if path exists it must be accessible and not a directory
	pathInfo, err := os.Stat(absPath)
	if err != nil {
		// ok if it doesn't exist
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return
	}
	if pathInfo.IsDir() {
		return fmt.Errorf("path '%s' is a directory", path)
	}

	return
}

func (lm *LogManager) SetPath(path string) (err error) {
	// if no path is specified default to stderr
	if path == "" {
		path = LOG_PATH_DEFAULT
	}

	switch path {
	case "stdout":
	case "stderr":
	default:
		err = checkLogPathValid(path)
	}

	if err != nil {
		lm.logPath = path
	}
	return
}

func (lm *LogManager) OpenLog() (err error) {
	// close existing log befoe opening a new one
	if err = lm.CloseLog(); err != nil {
		return
	}

	// open the appropriate log target
	switch lm.logPath {
	case "stdout":
		lm.logFile = os.Stdout
	case "stderr":
		lm.logFile = os.Stderr
	default:
		logFile, err := os.OpenFile(
			lm.logPath,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY,
			0600,
		)
		if err != nil {
			return err
		}
		lm.logFile = logFile
	}

	return
}

func (lm *LogManager) CloseLog() (err error) {
	// do nothing if no log file is open
	if lm.logFile == nil {
		return
	}

	switch lm.logPath {
	case "stdout":
		// nothing to do
	case "stderr":
		// nothing to do
	default:
		err = lm.logFile.Close()
	}

	// release the reference to the log file if no error occurred
	if err == nil {
		lm.logFile = nil
	}

	return
}

func (lm *LogManager) SetupHandler() (err error) {
	// handler options
	opts := &slog.HandlerOptions{Level: lm.logLevel}

	switch lm.logStyle {
	case "JSON":
		lm.logHandler = slog.NewJSONHandler(lm.logFile, opts)
	case "TEXT":
		lm.logHandler = slog.NewTextHandler(lm.logFile, opts)
	case "SYSLOG":
		return fmt.Errorf("support for '%s' style not yet implemented", lm.logStyle)
	default:
		return fmt.Errorf("'%s' style not supported", lm.logStyle)
	}
	return
}

func (lm *LogManager) Config(cfg *LogConfig) (err error) {
	// do nothing if no config is provided
	if cfg == nil {
		return
	}

	if err = lm.SetLevel(cfg.Level); err != nil {
		return
	}

	if err = lm.SetStyle(cfg.Style); err != nil {
		return
	}

	if err = lm.SetPath(cfg.Location); err != nil {
		return
	}

	return
}

func (lm *LogManager) Setup() (err error) {
	if err = lm.OpenLog(); err != nil {
		return
	}

	if err = lm.SetupHandler(); err != nil {
		return
	}

	// create a new logger using the new handler
	lm.logger = slog.New(lm.logHandler)

	// set our handler as the default slog handler
	slog.SetDefault(lm.logger)

	slog.Info(
		"Logging initialised",
		slog.Any("level", lm.logLevel),
		slog.String("dest", lm.logPath),
		slog.String("style", lm.logStyle),
	)

	return
}

func (lm *LogManager) ConfigAndSetup(cfg *LogConfig) (err error) {
	if err = lm.Config(cfg); err != nil {
		return err
	}

	return lm.Setup()
}

func SetupBasicLogging(debug bool) (err error) {
	lm := NewLogManager()
	if debug {
		lm.SetLevel("DEBUG")
	}
	return lm.Setup()
}
