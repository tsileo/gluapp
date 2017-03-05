package gluapp

import (
	"encoding/json"
	"errors"

	"github.com/yuin/gopher-lua"
)

// TableToMap convert a `*lua.LTable` to a `map[string]interface{}`
func tableToMap(table *lua.LTable) map[string]interface{} {
	res, _ := tomap(table, map[*lua.LTable]bool{})
	return res
}

func tomap(table *lua.LTable, visited map[*lua.LTable]bool) (map[string]interface{}, []interface{}) {
	res := map[string]interface{}{}
	var arrres []interface{}
	nkey := false
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		_, numberKey := key.(lua.LNumber)
		if numberKey {
			nkey = true
		}
		switch converted := value.(type) {
		case lua.LBool, lua.LNumber, lua.LString:
			if nkey {
				arrres = append(arrres, converted)
			} else {
				res[key.String()] = converted
			}
		case lua.LChannel:
			panic("no channel")
		case *lua.LFunction:
			panic("no function")
		case *lua.LNilType:
			res[key.String()] = converted
		case *lua.LState:
			panic("no LState")
		case *lua.LTable:
			var arr []interface{}
			obj := map[string]interface{}{}

			if visited[converted] {
				panic("nested table")
			}
			visited[converted] = true

			converted.ForEach(func(k lua.LValue, v lua.LValue) {
				_, numberKey := k.(lua.LNumber)
				// if numberKey, then convert to a slice of interface
				subtable, istable := v.(*lua.LTable)
				if numberKey {
					if istable {
						rtable, rarr := tomap(subtable, visited)
						if rarr != nil {
							arr = append(arr, rarr)
						} else {
							arr = append(arr, rtable)
						}
						// arr = append(arr, tomap(subtable, visited))
					} else {
						arr = append(arr, v)
					}
				} else {
					if istable {
						rtable, rarr := tomap(subtable, visited)
						if rarr != nil {
							obj[k.(lua.LString).String()] = rarr
						} else {
							obj[k.(lua.LString).String()] = rtable
						}
					} else {
						obj[k.(lua.LString).String()] = v
					}
				}
			})
			if len(arr) > 0 {
				if nkey {
					arrres = append(arrres, arr)
				} else {
					res[key.String()] = arr
				}
			} else {
				if nkey {
					arrres = append(arrres, obj)
				} else {
					res[key.String()] = obj
				}
			}
		}
	})
	return res, arrres
}

// Convert a Lua table to JSON
// Adapted from https://github.com/layeh/gopher-json/blob/master/util.go (Public domain)
func toJSON(value lua.LValue) []byte {
	var data []byte
	var err error
	switch converted := value.(type) {
	case lua.LBool:
		data, err = json.Marshal(converted)
	case lua.LChannel:
		err = errors.New("ToJSON: cannot marshal channel")
	case lua.LNumber:
		data, err = json.Marshal(converted)
	case *lua.LFunction:
		err = errors.New("ToJSON: cannot marshal function")
	case *lua.LNilType:
		data, err = json.Marshal(converted)
	case *lua.LState:
		err = errors.New("ToJSON: cannot marshal LState")
	case lua.LString:
		data, err = json.Marshal(converted)
	case *lua.LTable:
		data, err = json.Marshal(tableToMap(converted))
	case *lua.LUserData:
		err = errors.New("ToJSON: cannot marshal user data")
		// TODO: call metatable __tostring?
	}
	if err != nil {
		panic(err)
	}
	return data
}

// Convert the JSON to a Lua object ready to be pushed
// Adapted from https://github.com/layeh/gopher-json/blob/master/util.go (Public domain)
func fromJSON(L *lua.LState, js []byte) lua.LValue {
	var res interface{}
	if err := json.Unmarshal(js, &res); err != nil {
		panic(err)
	}
	return fromJSON2(L, res)
}

func interfaceToLValue(L *lua.LState, value interface{}) lua.LValue {
	return fromJSON2(L, value)
}

func fromJSON2(L *lua.LState, value interface{}) lua.LValue {
	switch converted := value.(type) {
	case bool:
		return lua.LBool(converted)
	case int:
		return lua.LNumber(converted)
	case int64:
		return lua.LNumber(converted)
	case float64:
		return lua.LNumber(converted)
	case string:
		return lua.LString(converted)
	case []interface{}:
		arr := L.CreateTable(len(converted), 0)
		for _, item := range converted {
			arr.Append(fromJSON2(L, item))
		}
		return arr
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(converted))
		for key, item := range converted {
			tbl.RawSetH(lua.LString(key), fromJSON2(L, item))
		}
		return tbl
	}
	return lua.LNil
}
