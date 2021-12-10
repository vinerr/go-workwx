package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xen0n/go-workwx/internal/lowlevel/encryptor"
	"github.com/xen0n/go-workwx/internal/lowlevel/envelope"
)

type LowLevelHandler struct {
	token     string
	encryptor *encryptor.WorkwxEncryptor
	ep        *envelope.Processor
	eh        EnvelopeHandler
	ctx       *gin.Context
}

var _ http.Handler = (*LowLevelHandler)(nil)

func NewLowLevelHandler(
	token string,
	encodingAESKey string,
	eh EnvelopeHandler,
) (*LowLevelHandler, error) {
	enc, err := encryptor.NewWorkwxEncryptor(encodingAESKey)
	if err != nil {
		return nil, err
	}

	ep, err := envelope.NewProcessor(token, encodingAESKey)
	if err != nil {
		return nil, err
	}

	return &LowLevelHandler{
		token:     token,
		encryptor: enc,
		ep:        ep,
		eh:        eh,
	}, nil
}

func (h *LowLevelHandler) SetGinContext(c *gin.Context) {
	h.ctx = c
}

func (h *LowLevelHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// 测试回调模式请求
		h.echoTestHandler(rw, r)

	case http.MethodPost:
		// 回调事件
		h.eventHandler(rw, r)

	default:
		// unhandled request method
		rw.WriteHeader(http.StatusNotImplemented)
	}
}
