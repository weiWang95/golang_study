package httpx

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type Response struct {
	Resp *http.Response

	parsed bool
	Body   []byte
}

func newResponse(resp *http.Response) *Response {
	r := &Response{
		Resp: resp,
	}

	return r
}

func (r *Response) Success() bool {
	return r.Resp.StatusCode < 300
}

func (r *Response) ContentType() string {
	return r.Resp.Header.Get("Content-Type")
}

func (r *Response) LoggerAble() bool {
	t := r.ContentType()
	return t == "application/json" || t == "text/plain"
}

func (r *Response) ParsedBytes() ([]byte, error) {
	if r.parsed {
		return r.Body, nil
	}

	defer r.Resp.Body.Close()

	data, err := ioutil.ReadAll(r.Resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body fail")
	}
	r.Body = data

	return r.Body, nil
}

func (r *Response) ParsedString() (string, error) {
	data, err := r.ParsedBytes()
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(data), nil
}

func (r *Response) ParsedStruct(v interface{}) error {
	data, err := r.ParsedBytes()
	if err != nil {
		return errors.WithStack(err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return errors.Wrap(err, "unmarshal fail")
	}

	return nil
}

func (r *Response) ParsedStream(w io.Writer) error {
	defer r.Resp.Body.Close()

	_, err := io.Copy(w, r.Resp.Body)
	if err != nil {
		return errors.Wrap(err, "read data fail")
	}

	return nil
}
