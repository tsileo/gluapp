# Gluapp

[![Build Status](https://travis-ci.org/tsileo/gluapp.svg?branch=master)](https://travis-ci.org/tsileo/gluapp)
&nbsp; &nbsp;[![Godoc Reference](https://godoc.org/a4.io/gluapp?status.svg)](https://godoc.org/a4.io/gluapp)

HTTP framework for [GopherLua](https://github.com/yuin/gopher-lua).

## Features

 - Simple
 - No 3rd party requirements except gopher-lua
 - Rely on Go template language
 - Same request/response idioms as Go HTTP lib
 - Comes with a basic (and optional) router
 - First-class JSON support
 - Included HTTP client

## Example

```lua
local router = require('router').new()

router:get('/hello/:name', function(params)
  gluapp.response:write('hello ' .. params.name)
end)

router:run()
```
