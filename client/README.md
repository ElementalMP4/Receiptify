# Receiptify Client

This is the client app, which is used to create templates. It also supports creating Lua plugins to generate content. Some example plusings have been provided. It's built using Go 1.24.1 and Fyne V2.

## Building and using

To build and use the UI, you'll need to install Go 1.24.1 (or newer) and then you can run it with `go run .` for testing, or package it using the Fyne CLI.

## Plugins

Plugins are written in Lua. I'm not very experienced with Lua so I cannot advise on complex plugin writing. You can have multiple Lua files in your plugin to make the code easier to read.

Plugins are only called by the Create UI, and are only run when the print button is pressed. Plugins can be called with this syntax:

```bash
{{PluginName.functionName("Param1", "Param2", "ParamN")}}
```

Parameters are optional. You specify your plugin functions in the plugin's manifest.json file. Something like this:

```json
{
  "name": "Greet",
  "version": "1.0.0",
  "functions": [
    {
      "name": "greet",
      "params": ["string"],
      "returns": ["string"]
    }
  ]
}

```

Plugins can receive and return multiple values. Note that multiple return values will simply be joined by spaces for now. The manifest here would allow you to call the greet function with:

```bash
{{Greet.greet("ElementalMP4")}}
```

The Lua code for this is in the example plugins directory. It's copied here as well for your convenience:

```lua
-- main.lua
Greet = {}

local helper = require("helper")

function Greet.greet(name)
    print("Getting ready to greet...")
    return helper.sayHi(name)
end
```

```lua
-- helper.lua
local helper = {}

function helper.sayHi(name)
    return "Hi, " .. name .. "!"
end

return helper
```

This code is not an efficient use of files, but is included to demonstrate how you can split your code into different files.