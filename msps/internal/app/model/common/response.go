package common

type Response struct {
	Success bool        `json:"success"` // 是否请求成功
	Msg     string      `json:"msg"`     // 请求结果描述
	Payload interface{} `json:"payload"` // 请求结果数据
}

type options struct {
	success bool
	msg     string
	payload interface{}	
}

type Option func(*options)

func WithSuccess(s bool) Option {
	return func(o *options) {
		o.success = s
	}
}

func WithMsg(msg string) Option {
	return func(o *options) {
		o.msg = msg
	}
}

func WithPayload(payload interface{}) Option {
	return func(o *options) {
		o.payload = payload
	}
}

func NewResponse(opts ...Option) Response {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	return Response{
		Success: o.success,
		Msg:     o.msg,
		Payload: o.payload,
	}
}
