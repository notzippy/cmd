// This package will be shared between Revel and Revel CLI eventually
package model

import (
	"github.com/revel/cmd/utils"
	"github.com/revel/config"
	"go/build"

	"errors"
	"fmt"
	"github.com/revel/cmd/logger"
	"path/filepath"
	"sort"
	"strings"
)

type (
	// The container object for describing all Revels variables
	RevelContainer struct {
		BuildPaths struct {
			Revel string
		}
		Paths struct {
			Import    string
			Source    string
			Base      string
			Code      []string              // Consolidated code paths
			Template  []string              // Consolidated template paths
			Config    []string              // Consolidated configuration paths
			ModuleMap map[string]*RevelUnit // The module path map
		}
		Info struct {
			Config   *config.Context // The global config
			Packaged bool            // True if packaged
			DevMode  bool            // True if devmode
			Vendor   bool            // True if vendored
			RunMode  string          // Set to the run mode
			Tests    bool            // True if test module included
		}
		Server struct {
			HTTPPort    int             // The http port
			HTTPAddr    string          // The http address
			HTTPSsl     bool            // True if running https
			HTTPSslCert string          // The SSL certificate
			HTTPSslKey  string          // The SSL key
			MimeConfig  *config.Context // The mime configuration

			CookiePrefix string // The cookie prefix
			CookieDomain string // The cookie domain
			CookieSecure bool   // True if cookie is secure
			SecretStr    string // The secret string
		}

		App   *RevelUnit    // The main app unit
		Units RevelUnitList // Additional module units (including revel)

		//ImportPath    string            // The import path
		//SourcePath    string            // The full source path
		//RunMode       string            // The current run mode
		//RevelPath     string            // The path to the Revel source code
		//BasePath      string            // The base path to the application
		//AppPath       string            // The application path (BasePath + "/app")
		//ViewsPath     string            // The application views path
		//CodePaths     []string          // All the code paths
		//TemplatePaths []string          // All the template paths
		//ConfPaths     []string          // All the configuration paths
		// Config        *config.Context   // The global config object
		//Packaged      bool              // True if packaged
		//DevMode       bool              // True if running in dev mode
		//HTTPPort      int               // The http port
		//HTTPAddr      string            // The http address
		//HTTPSsl       bool              // True if running https
		//HTTPSslCert   string            // The SSL certificate
		//HTTPSslKey    string            // The SSL key
		//AppName       string            // The application name
		//AppRoot       string            // The application root from the config `app.root`
	}
	RevelUnit struct {
		Name       string        // The friendly name for the unit
		Config     string        // The config file contents
		Type       RevelUnitType // The type of the unit
		Messages   string        // The messages
		BasePath   string        // The filesystem path of the unit
		ImportPath string        // The import path for the package
		Container  *RevelContainer
	}
	RevelUnitList []*RevelUnit
	RevelUnitType int

	WrappedRevelCallback struct {
		FireEventFunction func(key Event, value interface{}) (response EventResponse)
		ImportFunction    func(pkgName string) error
	}
)

const (
	APP    RevelUnitType = 1 // App always overrides all
	MODULE RevelUnitType = 2 // Module is next
	REVEL  RevelUnitType = 3 // Revel is last
)

// Simple Wrapped RevelCallback
func NewWrappedRevelCallback(fe func(key Event, value interface{}) (response EventResponse), ie func(pkgName string) error) RevelCallback {
	return &WrappedRevelCallback{fe, ie}
}

// Function to implement the FireEvent
func (w *WrappedRevelCallback) FireEvent(key Event, value interface{}) (response EventResponse) {
	if w.FireEventFunction != nil {
		response = w.FireEventFunction(key, value)
	}
	return
}
func (w *WrappedRevelCallback) PackageResolver(pkgName string) error {
	return w.ImportFunction(pkgName)
}

// RevelImportPath Revel framework import path
var RevelImportPath = "github.com/revel/revel"
var RevelModulesImportPath = "github.com/revel/modules"

// This function returns a container object describing the revel application
// eventually this type of function will replace the global variables.
func NewRevelPaths(mode, importPath string, callback RevelCallback) (rp *RevelContainer, err error) {
	log := utils.Logger.New("section", "logger", "importpath", importPath, "mode", mode)
	rp = &RevelContainer{}
	rp.Paths.ModuleMap = map[string]*RevelUnit{}
	// Ignore trailing slashes.

	// Add the App and Revel
	rp.App = rp.NewRevelUnit(APP, "", "", strings.TrimRight(importPath, "/"))
	var revelSourcePath string // Will be different from the app source path
	rp.App.BasePath, revelSourcePath, err = utils.FindSrcPaths(importPath, RevelImportPath, callback.PackageResolver)
	log.Info("Check source path", "app path", rp.App.BasePath, "revelpath", revelSourcePath, "error", err)
	if err != nil {
		return
	}
	// Add in the import path for the app
	rp.App.BasePath = filepath.Join(rp.App.BasePath, filepath.FromSlash(importPath))

	rp.Units.Add(rp.App)
	rp.Units.Add(rp.NewRevelUnit(REVEL, "revel", revelSourcePath, RevelImportPath))

	rp.Info.Vendor = utils.Exists(filepath.Join(rp.App.BasePath, "vendor"))

	// Sanity check , ensure app and conf paths exist
	if !utils.DirExists(rp.App.BasePath) {
		return rp, fmt.Errorf("No application found at path %s", rp.App.BasePath)
	}
	if !utils.DirExists(rp.App.GetConfigPath()) {
		return rp, fmt.Errorf("No configuration found at path %s", rp.App.GetConfigPath())
	}

	// Config load order
	// 1. framework (revel/conf/*)
	// 2. application (conf/*)
	// 3. user supplied configs (...) - User configs can override/add any from above
	// rp.ConfPaths = rp.Units.GetConfigPaths()

	rp.Info.Config, err = config.LoadContext("app.conf", rp.Units.GetConfigPaths())
	if err != nil {
		return rp, fmt.Errorf("Unable to load configuartion file %s", err)
	}
	rp.App.Name = rp.Info.Config.StringDefault("app.name", "(not set)")

	// Ensure that the selected runmode appears in app.conf.
	// If empty string is passed as the mode, treat it as "DEFAULT"
	if mode == "" {
		mode = config.DefaultSection
	}
	rp.Info.RunMode = mode

	if !rp.Info.Config.HasSection(mode) {
		return rp, fmt.Errorf("app.conf: No mode found: %s %s", "run-mode", mode)
	}
	rp.Info.Config.SetSection(mode)
	// Check for test mode
	rp.setTestMode()
	rp.Info.DevMode = rp.Info.Config.BoolDefault("mode.dev", false)

	// Configure Server properties from app.conf
	rp.Server.HTTPPort = rp.Info.Config.IntDefault("http.port", 9000)
	rp.Server.HTTPAddr = rp.Info.Config.StringDefault("http.addr", "")
	rp.Server.HTTPSsl = rp.Info.Config.BoolDefault("http.ssl", false)
	rp.Server.HTTPSslCert = rp.Info.Config.StringDefault("http.sslcert", "")
	rp.Server.HTTPSslKey = rp.Info.Config.StringDefault("http.sslkey", "")
	if rp.Server.HTTPSsl {
		if rp.Server.HTTPSslCert == "" {
			return rp, errors.New("No http.sslcert provided.")
		}
		if rp.Server.HTTPSslKey == "" {
			return rp, errors.New("No http.sslkey provided.")
		}
	}

	rp.Server.CookiePrefix = rp.Info.Config.StringDefault("cookie.prefix", "REVEL")
	rp.Server.CookieDomain = rp.Info.Config.StringDefault("cookie.domain", "")
	rp.Server.CookieSecure = rp.Info.Config.BoolDefault("cookie.secure", rp.Server.HTTPSsl)
	rp.Server.SecretStr = rp.Info.Config.StringDefault("app.secret", "")

	callback.FireEvent(REVEL_BEFORE_MODULES_LOADED, nil)
	if err := rp.loadModules(log, callback); err != nil {
		return rp, err
	}

	callback.FireEvent(REVEL_AFTER_MODULES_LOADED, nil)

	return
}

// LoadMimeConfig load mime-types.conf on init.
func (rp *RevelContainer) LoadMimeConfig() (err error) {
	rp.Server.MimeConfig, err = config.LoadContext("mime-types.conf", rp.Units.GetConfigPaths())
	if err != nil {
		return fmt.Errorf("Failed to load mime type config: %s %s", "error", err)
	}
	return
}

// Loads modules based on the configuration setup.
// This will fire the REVEL_BEFORE_MODULE_LOADED, REVEL_AFTER_MODULE_LOADED
// for each module loaded. The callback will receive the RevelContainer, name, moduleImportPath and modulePath
// It will automatically add in the code paths for the module to the
// container object
func (rp *RevelContainer) loadModules(log logger.MultiLogger, callback RevelCallback) (err error) {
	keys := []string{}
	for _, key := range rp.Info.Config.Options("module.") {
		keys = append(keys, key)
	}

	// Reorder module order by key name, a poor mans sort but at least it is consistent
	sort.Strings(keys)
	for _, key := range keys {
		moduleImportPath := rp.Info.Config.StringDefault(key, "")
		if moduleImportPath == "" {
			continue
		}

		modulePath, err := rp.ResolveImportPath(moduleImportPath)
		if err != nil {
			log.Info("Missing module ", "module_import_path", moduleImportPath, "error", err)
			callback.PackageResolver(moduleImportPath)
			modulePath, err = rp.ResolveImportPath(moduleImportPath)
			if err != nil {
				return fmt.Errorf("Failed to load module.  Import of path failed %s:%s %s:%s ", "modulePath", moduleImportPath, "error", err)
			}
		}
		// Drop anything between module.???.<name of module>
		name := key[len("module."):]
		if index := strings.Index(name, "."); index > -1 {
			name = name[index+1:]
		}
		callback.FireEvent(REVEL_BEFORE_MODULE_LOADED, []interface{}{rp, name, moduleImportPath, modulePath})
		rp.addModulePaths(name, moduleImportPath, modulePath)
		callback.FireEvent(REVEL_AFTER_MODULE_LOADED, []interface{}{rp, name, moduleImportPath, modulePath})
	}
	return
}

// Go through the list of modules and if one is the testrunner then set the flag
func (rp *RevelContainer) setTestMode() {
	for _, key := range rp.Info.Config.Options("module.") {
		if value := rp.Info.Config.StringDefault(key, ""); value == "github.com/revel/modules/testrunner" {
			rp.Info.Tests = true
		}
	}
	return
}

// Adds a module paths to the container object
func (rp *RevelContainer) addModulePaths(name, importPath, modulePath string) {
	module := rp.NewRevelUnit(MODULE, name, modulePath, importPath)
	rp.Units.Add(module)
}

// ResolveImportPath returns the filesystem path for the given import path.
// Returns an error if the import path could not be found.
func (rp *RevelContainer) ResolveImportPath(importPath string) (string, error) {
	if rp.Info.Packaged {
		return filepath.Join(rp.App.BasePath, importPath), nil
	}

	modPkg, err := build.Import(importPath, rp.App.BasePath, build.FindOnly)
	if err != nil {
		return "", err
	}
	if rp.Info.Vendor && !strings.HasPrefix(modPkg.Dir, rp.App.BasePath) {
		return "", fmt.Errorf("Module %s was found outside of path %s.", importPath, modPkg.Dir)
	}
	return modPkg.Dir, nil
}

// Returns the full list of code paths for the unit list
func (ul RevelUnitList) GetCodePaths() (list []string) {
	addTest := ul[0].Container.Info.Tests

	for _, r := range ul {
		if utils.DirExists(r.GetCodePath()) {
			list = append(list, r.GetCodePath())
		}
		if addTest {
			if utils.DirExists(r.GetTestPath()) {
				list = append(list, r.GetTestPath())
			}
		}
	}
	return
}

// Returns the first unit that matches this type
func (ul RevelUnitList) Get(unit RevelUnitType) (u *RevelUnit) {
	for _, r := range ul {
		if r.Type == unit {
			u = r
			return
		}
	}
	return
}

// Returns the full list of code paths for the unit list
func (ul RevelUnitList) GetConfigPaths() (list []string) {
	for _, r := range ul {
		if utils.DirExists(r.GetConfigPath()) {
			list = append(list, r.GetConfigPath())
		}
	}
	return
}

// Returns the full list of code paths for the unit list
func (ul RevelUnitList) GetViewPaths() (list []string) {
	for _, r := range ul {
		if utils.DirExists(r.GetViewPath()) {
			list = append(list, r.GetViewPath())
		}
	}
	return
}

// Returns the full list of code paths for the unit list
func (ul RevelUnitList) GetMessagePaths() (list []string) {
	for _, r := range ul {
		if utils.DirExists(r.GetMessagePath()) {
			list = append(list, r.GetMessagePath())
		}
	}
	return
}

// Append a unit to the list
func (ul *RevelUnitList) Add(unit *RevelUnit) {
	*ul = append(*ul, unit)
}

// Modules and apps foll
func (rc *RevelContainer) NewRevelUnit(unitType RevelUnitType, name, baseFilePath, importPath string) *RevelUnit {
	unit := &RevelUnit{
		Name:       name,
		Type:       unitType,
		BasePath:   baseFilePath,
		ImportPath: importPath,
		Container:  rc,
	}
	return unit
}

// Return the code path for the unit
func (u *RevelUnit) GetCodePath() string {
	return filepath.Join(u.BasePath, "app")
}

// Return the test code path for the unit
func (u *RevelUnit) GetTestPath() string {
	return filepath.Join(u.BasePath, "tests")
}

// Return the view path for the unit
func (u *RevelUnit) GetViewPath() string {
	return filepath.Join(u.BasePath, "app", "views")
}

// Return the config path for the unit
func (u *RevelUnit) GetConfigPath() string {
	return filepath.Join(u.BasePath, "conf")
}

// Return the message path for the unit
func (u *RevelUnit) GetMessagePath() string {
	return filepath.Join(u.BasePath, "messages")
}
