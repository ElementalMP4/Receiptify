greet = {}

local helper = require("helper")

function greet.greet(name)
    print("Getting ready to greet...")
    return helper.sayHi(name)
end
