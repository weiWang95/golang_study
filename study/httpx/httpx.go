package httpx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Httpx struct {
	logger *logrus.Logger
	cfg    *Config

	beforeRequest RequestHook
}

func NewHttpx(fns ...configFn) *Httpx {
	cfg := defaultCfg()
	for _, fn := range fns {
		fn(cfg)
	}

	return &Httpx{
		logger: logrus.New(),
		cfg:    cfg,
	}
}

func GET(ctx context.Context, url string, query map[string]interface{}) (*Response, error) {
	return NewHttpx().GET(ctx, url, query)
}

func POST(ctx context.Context, url string, opt *Option) (*Response, error) {
	return NewHttpx().POST(ctx, url, opt)
}

func PUT(ctx context.Context, url string, opt *Option) (*Response, error) {
	return NewHttpx().PUT(ctx, url, opt)
}

func PATCH(ctx context.Context, url string, opt *Option) (*Response, error) {
	return NewHttpx().PATCH(ctx, url, opt)
}

func DELETE(ctx context.Context, url string, opt *Option) (*Response, error) {
	return NewHttpx().DELETE(ctx, url, opt)
}

func (h *Httpx) GET(ctx context.Context, url string, query map[string]interface{}) (*Response, error) {
	return h.Request(ctx, http.MethodGet, url, &Option{Query: query})
}

func (h *Httpx) POST(ctx context.Context, url string, opt *Option) (*Response, error) {
	return h.Request(ctx, http.MethodPost, url, opt)
}

func (h *Httpx) PUT(ctx context.Context, url string, opt *Option) (*Response, error) {
	return h.Request(ctx, http.MethodPut, url, opt)
}

func (h *Httpx) PATCH(ctx context.Context, url string, opt *Option) (*Response, error) {
	return h.Request(ctx, http.MethodPatch, url, opt)
}

func (h *Httpx) DELETE(ctx context.Context, url string, opt *Option) (*Response, error) {
	return h.Request(ctx, http.MethodDelete, url, opt)
}

func (h *Httpx) Request(ctx context.Context, method string, url string, opt *Option) (*Response, error) {
	req, err := h.buildRequest(ctx, method, url, opt)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cli := http.Client{}
	if h.cfg.Timeout != 0 {
		cli.Timeout = h.cfg.Timeout
	}

	if opt != nil && opt.Params != nil {
		h.logger.Infof("[%s] %s %+v", method, req.Req.URL.String(), opt.Params)
	} else {
		h.logger.Infof("[%s] %s", method, req.Req.URL.String())
	}

	resp, err := cli.Do(req.Req)
	if err != nil {
		return nil, errors.Wrap(err, "do request fail")
	}

	r := newResponse(resp)

	if r.LoggerAble() {
		respBody, _ := r.ParsedString()
		h.logger.Debugf("%d -> %s", r.Resp.StatusCode, respBody)
	}

	if !r.Success() {
		respBody, _ := r.ParsedString()
		return r, errors.Errorf("code: %d, msg: %s", r.Resp.StatusCode, respBody)
	}

	return r, nil
}

func (h *Httpx) buildRequest(ctx context.Context, method string, url string, opt *Option) (*Request, error) {
	reqUrl := h.getRequestUrl(url)

	req, err := newRequest(ctx, method, reqUrl, opt)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	h.runHook(req)

	return req, nil
}

func (h *Httpx) getRequestUrl(url string) string {
	if h.cfg.Host == "" {
		return url
	}

	return fmt.Sprintf("%s%s%s", h.cfg.Host, h.cfg.UrlPrefix, url)
}

func (h *Httpx) SetBeforeRequestHook(fn func(req *Request)) {
	h.beforeRequest = fn
}

func (h *Httpx) runHook(req *Request) {
	if h.beforeRequest != nil {
		h.beforeRequest(req)
	}
}
