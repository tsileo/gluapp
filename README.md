# Gluapp

[![Build Status](https://travis-ci.org/tsileo/gluapp.svg?branch=master)](https://travis-ci.org/tsileo/gluapp)
&nbsp; &nbsp;[![Godoc Reference](https://godoc.org/a4.io/gluapp?status.svg)](https://godoc.org/a4.io/gluapp)

HTTP framework for [GopherLua](https://github.com/yuin/gopher-lua).

lua
```
local router = require('router').new()

router:get('/hello/:name', function(params)
  response:write('hello ' .. params.name)
end)

router:run()

```
