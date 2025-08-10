package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

var (
	manifests map[string]*PluginManifest
	luaVm     *lua.LState
)

func ConfigureLuaAndLoadPlugins() error {
	luaVm = nil
	manifests = make(map[string]*PluginManifest)

	if settings.PluginPath == "" {
		return fmt.Errorf("plugin path not set")
	}

	luaVm = lua.NewState()
	pluginFolders, err := os.ReadDir(settings.PluginPath)
	if err != nil {
		return err
	}

	// Load plugins and manifests
	for _, folder := range pluginFolders {
		if folder.IsDir() {
			pluginPath := filepath.Join(settings.PluginPath, folder.Name())
			fmt.Println("Loading plugin from folder:", folder.Name())

			manifestPath := filepath.Join(pluginPath, "manifest.json")
			manifestData, err := os.ReadFile(manifestPath)
			if err != nil {
				return fmt.Errorf("error reading manifest for %s: %v", folder.Name(), err)
			}

			var manifest PluginManifest
			if err := json.Unmarshal(manifestData, &manifest); err != nil {
				return fmt.Errorf("error parsing manifest for %s: %v", folder.Name(), err)
			}
			manifests[folder.Name()] = &manifest

			// Set package.path to plugin folder so we can use require()
			pluginLuaPath := filepath.Join(pluginPath, "?.lua")
			packagePathSetLua := fmt.Sprintf(`package.path = "%s;" .. package.path`, pluginLuaPath)
			if err := luaVm.DoString(packagePathSetLua); err != nil {
				return fmt.Errorf("failed to set package.path for plugin %s: %v", folder.Name(), err)
			}

			// Load main.lua
			mainLuaPath := filepath.Join(pluginPath, "main.lua")
			if err := luaVm.DoFile(mainLuaPath); err != nil {
				return fmt.Errorf("failed to load main.lua for plugin %s: %v", folder.Name(), err)
			}

			fmt.Printf("Successfully loaded %s %s from %s\n", manifest.PluginName, manifest.Version, folder.Name())
		}
	}

	return nil
}

func RunPlugin(token string) ([]string, error) {
	callStr := strings.TrimSuffix(strings.TrimPrefix(token, "{{"), "}}")
	dotIndex := strings.Index(callStr, ".")
	if dotIndex < 1 {
		return []string{}, fmt.Errorf("call must be pluginName.functionName")
	}

	pluginName := callStr[:dotIndex]
	funcCall := callStr[dotIndex+1:]

	manifest, exists := manifests[pluginName]
	if !exists {
		return []string{}, fmt.Errorf("plugin %s not found", pluginName)
	}

	parenIndex := strings.Index(funcCall, "(")
	if parenIndex == -1 || !strings.HasSuffix(funcCall, ")") {
		return []string{}, fmt.Errorf("function call is missing parentheses")
	}
	funcName := funcCall[:parenIndex]
	argsStr := funcCall[parenIndex+1 : len(funcCall)-1]

	// Find function info in manifest
	var funcInfo *FunctionInfo
	for _, f := range manifest.Functions {
		if f.Name == funcName {
			funcInfo = &f
			break
		}
	}
	if funcInfo == nil {
		return []string{}, fmt.Errorf("function %s not found in %s manifest", funcName, pluginName)
	}

	// Get plugin table from Lua
	pluginTable := luaVm.GetGlobal(pluginName)
	if pluginTable == lua.LNil {
		return []string{}, fmt.Errorf("plugin %s not found", pluginName)
	}
	pluginTableTable, ok := pluginTable.(*lua.LTable)
	if !ok {
		return []string{}, fmt.Errorf("plugin %s is not a Lua table", pluginName)
	}

	fn := luaVm.GetField(pluginTableTable, funcName)
	if fn == lua.LNil {
		return []string{}, fmt.Errorf("function %s not found in plugin %s", funcName, pluginName)
	}

	// Parse arguments
	args := []lua.LValue{}
	argsStr = strings.TrimSpace(argsStr)
	if len(argsStr) > 0 {
		argParts := splitArgs(argsStr)
		if len(argParts) != len(funcInfo.Params) {
			return []string{}, fmt.Errorf("expected %d args, got %d", len(funcInfo.Params), len(argParts))
		}

		for i, p := range argParts {
			p = strings.TrimSpace(p)
			switch funcInfo.Params[i] {
			case "string":
				if len(p) >= 2 && p[0] == '"' && p[len(p)-1] == '"' {
					args = append(args, lua.LString(p[1:len(p)-1]))
				} else {
					return []string{}, fmt.Errorf("string args must be in quotes")
				}
			case "number":
				var num float64
				_, err := fmt.Sscanf(p, "%f", &num)
				if err != nil {
					return []string{}, fmt.Errorf("failed to parse number arg: %s", p)
				}
				args = append(args, lua.LNumber(num))
			case "boolean":
				switch p {
				case "true":
					args = append(args, lua.LBool(true))
				case "false":
					args = append(args, lua.LBool(false))
				default:
					return []string{}, fmt.Errorf("boolean args must be true or false")
				}
			default:
				return []string{}, fmt.Errorf("unsupported param type: %s", funcInfo.Params[i])
			}
		}
	}

	// Call function with correct number of returns
	numRets := len(funcInfo.Returns)
	err := luaVm.CallByParam(lua.P{
		Fn:      fn,
		NRet:    numRets,
		Protect: true,
	}, args...)
	if err != nil {
		return []string{}, fmt.Errorf("error calling Lua function: %v", err)
	}

	// Collect returns
	rets := make([]string, 0, numRets)
	for i := numRets; i >= 1; i-- {
		ret := luaVm.Get(-i)
		typ := funcInfo.Returns[numRets-i]
		switch typ {
		case "string":
			if str, ok := ret.(lua.LString); ok {
				rets = append(rets, string(str))
			} else {
				rets = append(rets, ret.String())
			}
		case "number":
			if num, ok := ret.(lua.LNumber); ok {
				rets = append(rets, fmt.Sprintf("%f", float64(num)))
			} else {
				rets = append(rets, ret.String())
			}
		case "boolean":
			if b, ok := ret.(lua.LBool); ok {
				rets = append(rets, fmt.Sprintf("%t", bool(b)))
			} else {
				rets = append(rets, ret.String())
			}
		default:
			rets = append(rets, ret.String())
		}
	}
	luaVm.Pop(numRets)

	return rets, nil
}

func splitArgs(s string) []string {
	var args []string
	inQuotes := false
	escaped := false
	start := 0

	for i, c := range s {
		if c == '\\' && !escaped {
			escaped = true
			continue
		}
		if c == '"' && !escaped {
			inQuotes = !inQuotes
		}
		if c == ',' && !inQuotes {
			args = append(args, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
		escaped = false
	}
	args = append(args, strings.TrimSpace(s[start:]))

	return args
}
