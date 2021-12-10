package httpapi

import (
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/xen0n/go-workwx/internal/lowlevel/envelope"
)

type EnvelopeHandler interface {
	OnIncomingEnvelope(ctx *gin.Context, rx envelope.Envelope) error
}

func (h *LowLevelHandler) eventHandler(
	rw http.ResponseWriter,
	r *http.Request,
) {
	// request bodies are assumed small
	// we can't do streaming parse/decrypt/verification anyway
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	logrus.Debugln(r.Method)

	// signature verification is inside EnvelopeProcessor
	ev, err := h.ep.HandleIncomingMsg(r.URL, body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	logrus.Debugln(r.Method)

	err = h.eh.OnIncomingEnvelope(h.ctx, ev)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	logrus.Debugln(r.Method)

	// currently we always return empty 200 responses
	// any reply is to be sent asynchronously
	// this might change in the future (maybe save a couple of RTT or so)
	if h.ctx == nil {
		rw.WriteHeader(http.StatusOK)
	}
}
