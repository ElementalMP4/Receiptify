Greet = {}

local helper = require("helper")

function Greet.greet(name)
    print("Getting ready to greet...")
    return helper.sayHi(name)
end
