package workwx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xen0n/go-workwx/internal/lowlevel/envelope"
	"github.com/xen0n/go-workwx/internal/lowlevel/httpapi"
)

// RxMessageHandler 用来接收消息的接口。
type RxMessageHandler interface {
	// OnIncomingMessage 一条消息到来时的回调。
	OnIncomingMessage(ctx *gin.Context, msg *RxMessage) error
}

type lowlevelEnvelopeHandler struct {
	highlevelHandler RxMessageHandler
}

var _ httpapi.EnvelopeHandler = (*lowlevelEnvelopeHandler)(nil)

func (h *lowlevelEnvelopeHandler) OnIncomingEnvelope(ctx *gin.Context, rx envelope.Envelope) error {
	msg, err := fromEnvelope(rx.Msg)
	if err != nil {
		return err
	}
	return h.highlevelHandler.OnIncomingMessage(ctx, msg)
}

type HTTPHandler struct {
	ctx   *gin.Context
	inner *httpapi.LowLevelHandler
}

var _ http.Handler = (*HTTPHandler)(nil)

func NewHTTPHandler(
	token string,
	encodingAESKey string,
	rxMessageHandler RxMessageHandler,
) (*HTTPHandler, error) {
	lleh := &lowlevelEnvelopeHandler{
		highlevelHandler: rxMessageHandler,
	}

	llHandler, err := httpapi.NewLowLevelHandler(token, encodingAESKey, lleh)
	if err != nil {
		return nil, err
	}

	obj := HTTPHandler{
		inner: llHandler,
	}

	return &obj, nil
}

func (h *HTTPHandler) SetGinContext(c *gin.Context) {
	h.ctx = c
	h.inner.SetGinContext(h.ctx)
}

func (h *HTTPHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	h.inner.ServeHTTP(rw, r)
}
