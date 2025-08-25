package strinterplot

import (
	"encoding/json"
	"fmt"
	"strconv"

	var_template "github.com/xhd2015/go-var-template"
)

func InterplotList(list []string, args map[string]any) ([]string, error) {
	argsStr := make(map[string]string, len(args))
	for k, v := range args {
		str, err := getStr(v)
		if err != nil {
			return nil, fmt.Errorf("get str %s: %v", k, err)
		}
		argsStr[k] = str
	}

	res := make([]string, len(list))
	for i, v := range list {
		str, err := interplot(v, argsStr)
		if err != nil {
			return nil, fmt.Errorf("interplot %s: %v", v, err)
		}
		res[i] = str
	}
	return res, nil
}

func interplot(tpl string, args map[string]string) (string, error) {
	ctpl := var_template.Compile(tpl)
	return ctpl.Execute(args)
}

func getStr(v interface{}) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	}
	jsonRes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonRes), nil
}
