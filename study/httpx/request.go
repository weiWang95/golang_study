package httpx

import (
	"bytes"
	"context"
	"net/http"

	"github.com/pkg/errors"
)

type Request struct {
	Req *http.Request
}

func newRequest(ctx context.Context, method string, url string, opt *Option) (*Request, error) {
	var reader bytes.Reader

	if opt != nil {
		b, err := opt.Parsed()
		if err != nil {
			return nil, errors.Wrap(err, "parsed params fail")
		}
		reader.Read(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, &reader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	r := &Request{
		Req: req,
	}

	if opt != nil {
		opt.ParseQuery(r)
	}

	return r, nil
}
