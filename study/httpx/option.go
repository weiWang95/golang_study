package httpx

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Option struct {
	Query  map[string]interface{}
	Params interface{}
}

func (o *Option) Parsed() ([]byte, error) {
	if o.Params == nil {
		return nil, nil
	}

	switch reflect.TypeOf(o.Params).Kind() {
	case reflect.String:
		return []byte(o.Params.(string)), nil
	default:
		b, err := json.Marshal(o.Params)
		return b, err
	}
}

func (o *Option) ParseQuery(req *Request) {
	if o.Query == nil {
		return
	}

	q := req.Req.URL.Query()

	for k, v := range o.Query {
		t := reflect.TypeOf(v)
		switch t.Kind() {
		case reflect.Array, reflect.Slice:
			arr := reflect.ValueOf(v)
			for i := 0; i < arr.Len(); i++ {
				q.Add(k, fmt.Sprint(arr.Index(i)))
			}
		default:
			q.Add(k, fmt.Sprint(v))
		}
	}

	req.Req.URL.RawQuery = q.Encode()
}
