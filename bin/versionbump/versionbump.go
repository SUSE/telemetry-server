package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	tutils "github.com/SUSE/telemetry/pkg/utils"
)

// version string elements
const (
	VERSION_PREFIX            = "v"
	VERSION_BUILD_JOINER      = "+"
	VERSION_PRERELEASE_JOINER = "-"
	VERSION_FIELD_JOINER      = "."
)

type CmdMode string

// bump modes
const (
	CMD_BUMP_MAJOR = CmdMode("major")
	CMD_BUMP_MINOR = CmdMode("minor")
	CMD_BUMP_PATCH = CmdMode("patch")

	CMD_UPDATE = CmdMode("update")
)

var cmdModes = []string{
	string(CMD_BUMP_MAJOR),
	string(CMD_BUMP_MINOR),
	string(CMD_BUMP_PATCH),
	string(CMD_UPDATE),
}

func (c *CmdMode) String() string {
	return string(*c)
}

func (c *CmdMode) Valid() bool {
	switch *c {
	case CMD_BUMP_MAJOR:
		fallthrough
	case CMD_BUMP_MINOR:
		fallthrough
	case CMD_BUMP_PATCH:
		fallthrough
	case CMD_UPDATE:
		return true
	}
	return false
}

func CommandName() string {
	return filepath.Base(os.Args[0])
}

func AppVersionPath() string {
	return filepath.Join("app", "VERSION")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	} else if !errors.Is(err, os.ErrNotExist) {
		warningMsg("Failed to os.Stat(%q): %w", path, err)
	}
	return false
}

func CurrentDirectory() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("failed to get current directory: %w", err)
		return "", err
	}
	return wd, nil
}

func GetVersionFilePath() (verPath string, err error) {
	// the path to version file from the top of the code base
	appVerPath := AppVersionPath()
	debugMsg("Version File Relative Path: %q", appVerPath)

	// list of possible code base top level paths under which to look for version file
	codeBasePaths := []string{}

	// add the current directory as a search path
	currDir, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("failed to get current directory: %w", err)
		return
	}
	debugMsg("currDir: %q", currDir)
	codeBasePaths = append(codeBasePaths, currDir)

	// if running the built binary from it's directory under the code base
	cmdPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		err = fmt.Errorf("failed to determine command path from %q (os.Args[0]): %w", os.Args[0], err)
		return
	}
	debugMsg("cmdPath: %q", cmdPath)
	codeBasePaths = append(codeBasePaths, filepath.Dir(filepath.Dir(cmdPath)))

	// if running the binary from somewhere else, e.g. from a temp directory
	// when using go run, then attempt to use the source code path built into
	// the binary itself
	_ /* pc */, srcFile, _ /* line */, ok := runtime.Caller(0)
	if !ok {
		err = fmt.Errorf("failed to determine source file path")
		return
	}
	debugMsg("srcFile: %q", srcFile)
	codeBasePaths = append(codeBasePaths, filepath.Dir(filepath.Dir(filepath.Dir(srcFile))))

	debugMsg("Potential Version File Paths: %+v", codeBasePaths)
	for _, cbPath := range codeBasePaths {
		chkPath := filepath.Join(cbPath, appVerPath)
		if pathExists(chkPath) {
			verPath = chkPath
		}
	}

	if verPath == "" {
		err = fmt.Errorf("unabled to find version file %q", appVerPath)
		return
	}

	debugMsg("Version File Path: %q", verPath)
	return
}

type ParsedVersion struct {
	Major, Minor, Patch uint64
	Prerelease, Build   string
}

func NewParsedVersion(version string) (*ParsedVersion, error) {
	p := new(ParsedVersion)
	if err := p.Parse(version); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *ParsedVersion) String() string {
	var ver []string
	ver = append(ver, fmt.Sprintf("v%d.%d.%d", p.Major, p.Minor, p.Patch))
	if p.Prerelease != "" {
		ver = append(ver, "-", p.Prerelease)
	}
	if p.Build != "" {
		ver = append(ver, "+", p.Build)
	}
	return strings.Join(ver, "")
}

func (p *ParsedVersion) Parse(version string) (err error) {
	var prerelease, build string
	var major, minor, patch uint64

	ver := version
	if !strings.HasPrefix(ver, VERSION_PREFIX) {
		return fmt.Errorf("expected version %q to start with a %q", ver, VERSION_PREFIX)
	}

	buildSplit := strings.Split(ver, VERSION_BUILD_JOINER)
	if len(buildSplit) > 1 {
		build = buildSplit[1]
	}
	ver = buildSplit[0]

	dashSplit := strings.Split(ver, VERSION_PRERELEASE_JOINER)
	if len(dashSplit) > 1 {
		prerelease = dashSplit[1]
	}
	ver = dashSplit[0]

	verSplit := strings.Split(ver, VERSION_FIELD_JOINER)
	switch len(verSplit) {
	case 3:
		if patch, err = parseUint64(verSplit[2]); err != nil {
			return err
		}
		fallthrough
	case 2:
		if minor, err = parseUint64(verSplit[1]); err != nil {
			return err
		}
		fallthrough
	case 1:
		// skip the v prefix in the major field
		if major, err = parseUint64(verSplit[0][1:]); err != nil {
			return err
		}
	default:
		return fmt.Errorf("version %q doesn't match format v<Major>[.<Minor>[.<Patch>]][-<Prerelease>][+Build]", version)
	}

	// successfully parsed provided version
	p.Major = major
	p.Minor = minor
	p.Patch = patch
	p.Prerelease = prerelease
	p.Build = build

	return
}

func (p *ParsedVersion) reportFieldUpdate(field string, value any) {
	if value != "" {
		verboseMsg("Setting %s to %v", field, value)
	} else {
		verboseMsg("Clearing %s", field)
	}
}

func (p *ParsedVersion) UpdatePrerelease(prerelease string) {
	p.reportFieldUpdate("prerelease", prerelease)
	p.Prerelease = prerelease
}

func (p *ParsedVersion) UpdateBuild(build string) {
	p.reportFieldUpdate("build", build)
	p.Build = build
}

func (p *ParsedVersion) VersionBump(bumpMode CmdMode) {
	// update version fields appropriately
	switch bumpMode {
	case CMD_BUMP_MAJOR:
		p.Major++
		p.reportFieldUpdate("major", p.Major)
		p.Minor = 0
		p.reportFieldUpdate("minor", p.Minor)
		p.Patch = 0
		p.reportFieldUpdate("patch", p.Patch)
	case CMD_BUMP_MINOR:
		p.Minor++
		p.reportFieldUpdate("minor", p.Minor)
		p.Patch = 0
		p.reportFieldUpdate("patch", p.Patch)
	case CMD_BUMP_PATCH:
		p.Patch++
		p.reportFieldUpdate("patch", p.Patch)
	default:
		log.Fatalf("Invalid bump field type %q", bumpMode)
	}
}

func parseUint64(number string) (value uint64, err error) {
	value, err = strconv.ParseUint(number, 0, 64)
	if err != nil {
		err = fmt.Errorf("failed to parse %q as a uint64: %w", number, err)
	}
	return
}

type VersionBumpOpts struct {
	Mode       CmdMode
	Prerelease string
	Build      string
	DryRun     bool
}

var opt_defaults = VersionBumpOpts{
	Mode:       "patch",
	Prerelease: "",
	Build:      "",
	DryRun:     false,
}

func usage() {
	fmt.Fprintf(
		flag.CommandLine.Output(),
		"Usage: %s %s %s MODE\n%s\n%s\n%s\n",
		CommandName(),
		"[-b|--build BUILD]",
		"[-p|--prerelease PRERELEASE]",
		"Where:",
		"  MODE",
		"\tMODE must be one of ["+strings.Join(cmdModes, ", ")+"]",
	)
	flag.PrintDefaults()
}

func parseArgs() *VersionBumpOpts {
	opts := new(VersionBumpOpts)

	// dryrun
	for _, f := range []string{"d", "dryrun"} {
		flag.BoolVar(&opts.DryRun, f, opt_defaults.DryRun, "Report actions that would be taken without performing them, implies verbose.")
	}

	// verbose
	for _, f := range []string{"v", "verbose"} {
		flag.BoolVar(&verbose, f, false, "Report actions that are performed.")
	}

	// debug, special case, using global variable
	for _, f := range []string{"D", "debug"} {
		flag.BoolVar(&debug, f, false, "Enable debugging output.")
	}

	// prerelease
	for _, f := range []string{"p", "prerelease"} {
		flag.StringVar(&opts.Prerelease, f, opt_defaults.Prerelease, "`PRERELEASE` value to use when generating version")
	}

	// build
	for _, f := range []string{"b", "build"} {
		flag.StringVar(&opts.Build, f, opt_defaults.Build, "`BUILD` value to use when generating version")
	}

	// parse the provided arguments using custom usage message
	flag.Usage = usage
	flag.Parse()

	// if dryrun was specified, ensure verbose is true as well
	if opts.DryRun {
		verbose = true
		debugMsg("dryrun mode enabled, versbose set to true")
	}

	// determine the mode
	if flag.NArg() < 1 {
		errorMsg("No MODE specified")
		flag.Usage()
		os.Exit(1)
	}

	debugMsg("flag.Arg(1): %q", flag.Arg(0))

	opts.Mode = CmdMode(flag.Arg(0))
	if !opts.Mode.Valid() {
		errorMsg("Invalid MODE %q specified. Must be one of ["+strings.Join(cmdModes, ", ")+"]", opts.Mode)
		flag.Usage()
		os.Exit(1)
	}

	debugMsg("Options: %+v", opts)

	return opts
}

type VersionFileManager struct {
	verFile tutils.FileManager
	verTemp tutils.FileManager
	Version *ParsedVersion
}

func NewVersionFile(verPath string) (verFile *VersionFileManager, err error) {
	verTemp := verPath + ".tmp"
	vfm := new(VersionFileManager)

	// setup the managed version file
	vfm.verFile = tutils.NewManagedFile()
	defer vfm.verFile.Close()
	err = vfm.verFile.UseExistingFile(verPath)
	if err != nil {
		err = fmt.Errorf("failed to setup access to version file %q: %w", verPath, err)
		return
	}
	vfm.verFile.DisableBackups()

	// setup the managed temporary version file
	vfm.verTemp = tutils.NewManagedFile()
	defer vfm.verTemp.Close()
	err = vfm.verTemp.Init(verTemp, vfm.verFile.User(), vfm.verFile.Group(), vfm.verFile.Perm())
	if err != nil {
		err = fmt.Errorf("failed to setup access to temporary version file %q: %w", verTemp, err)
		return
	}
	vfm.verTemp.DisableBackups()

	// retrieve the version
	err = vfm.verFile.Open(false)
	if err != nil {
		err = fmt.Errorf("failed to open version file %q: %w", vfm.verFile.Path(), err)
		return
	}
	verBytes, err := vfm.verFile.Read()
	if err != nil {
		err = fmt.Errorf("failed to read version file %q: %w", vfm.verFile.Path(), err)
		return
	}

	version := strings.TrimSpace(string(verBytes))
	vfm.Version, err = NewParsedVersion(version)
	if err != nil {
		err = fmt.Errorf("failed to parse %q as a version: %w", version, err)
		return
	}

	// setup complete
	verFile = vfm

	return
}

func (vfm *VersionFileManager) String() string {
	return vfm.Version.String()
}

func (vfm *VersionFileManager) Process(opts *VersionBumpOpts) {
	verboseMsg("Existing version is %q", vfm.Version)
	if opts.Mode != CMD_UPDATE {
		fmt.Printf("Bumping %q value in version\n", opts.Mode)
		vfm.Version.VersionBump(opts.Mode)
	}

	vfm.Version.UpdatePrerelease(opts.Prerelease)
	vfm.Version.UpdateBuild(opts.Build)

	verboseMsg("Updating version to %q", vfm.Version)
}

func (vfm *VersionFileManager) Update() (err error) {
	// create the temporary version file, and if successful, setup a deferred delete of it
	err = vfm.verTemp.Create()
	if err != nil {
		err = fmt.Errorf("failed to create temporary version file %q: %w", vfm.verTemp.Path(), err)
		return
	}
	defer vfm.verTemp.Delete()

	// render updated version string and write it out to the temporary version file
	updatedBytes := []byte(fmt.Sprintf("%s\n", vfm.Version))
	err = vfm.verTemp.Update(updatedBytes)
	if err != nil {
		err = fmt.Errorf("failed to write updated version %q to temporary version file %q: %w", vfm.Version, vfm.verTemp.Path(), err)
		return
	}

	// move the temporary version file to replace to original version file
	err = os.Rename(vfm.verTemp.Path(), vfm.verFile.Path())
	if err != nil {
		err = fmt.Errorf("failed to move temporary version file %q to replace version file %q: %w", vfm.verTemp.Path(), vfm.verFile.Path(), err)
		return
	}

	return
}

var debug bool

func debugMsg(msgFmt string, msgArgs ...any) {
	if !debug {
		return
	}
	fmt.Fprintf(os.Stderr, "DEBUG: "+msgFmt+"\n", msgArgs...)
}

var verbose bool

func verboseMsg(msgFmt string, msgArgs ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, msgFmt+"\n", msgArgs...)
}

func warningMsg(msgFmt string, msgArgs ...any) {
	fmt.Fprintf(os.Stderr, "WARNING: "+msgFmt+"\n", msgArgs...)
}

func errorMsg(msgFmt string, msgArgs ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+msgFmt+"\n", msgArgs...)
}

func main() {
	opts := parseArgs()

	versionFile, err := GetVersionFilePath()
	if err != nil {
		errorMsg("Failed to determine version file path: %s", err)
		flag.Usage()
		os.Exit(1)
	}

	vfMgr, err := NewVersionFile(versionFile)
	if err != nil {
		errorMsg("Failed to create version file manager: %s", err)
		flag.Usage()
		os.Exit(1)
	}

	debugMsg("Existing Version: %q", vfMgr.Version)

	vfMgr.Process(opts)

	if !opts.DryRun {
		err = vfMgr.Update()
		if err != nil {
			errorMsg("Failed to update version file %q: %s", versionFile, err)
			flag.Usage()
			os.Exit(1)
		}
	} else {
		fmt.Printf("DRYRUN: Version updates not saved\n")
	}
}
