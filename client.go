package workwx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/url"
	"sync"
	"time"

	"github.com/gocolly/colly"
	"github.com/sirupsen/logrus"
)

// Chrome相关
const (
	TimeZone           = "Asia/Shanghai"
	UserAgentForChrome = `Mozilla/5.0 (Windows NT  6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`
)

// Workwx 企业微信客户端
type Workwx struct {
	opts options

	// CorpID 企业 ID，必填
	CorpID string
}

// WorkwxApp 企业微信客户端（分应用）
type WorkwxApp struct {
	*Workwx

	// CorpSecret 应用的凭证密钥，必填
	CorpSecret string
	// AgentID 应用 ID，必填
	AgentID                int64
	accessToken            *token
	jsapiTicket            *token
	jsapiTicketAgentConfig *token
}

// New 构造一个 Workwx 客户端对象，需要提供企业 ID
func New(corpID string, opts ...CtorOption) *Workwx {
	optionsObj := defaultOptions()

	for _, o := range opts {
		o.applyTo(&optionsObj)
	}

	return &Workwx{
		opts: optionsObj,

		CorpID: corpID,
	}
}

// WithApp 构造本企业下某自建 app 的客户端
func (c *Workwx) WithApp(corpSecret string, agentID int64) *WorkwxApp {
	app := WorkwxApp{
		Workwx: c,

		CorpSecret: corpSecret,
		AgentID:    agentID,

		accessToken:            &token{mutex: &sync.RWMutex{}},
		jsapiTicket:            &token{mutex: &sync.RWMutex{}},
		jsapiTicketAgentConfig: &token{mutex: &sync.RWMutex{}},
	}
	app.accessToken.setGetTokenFunc(app.getAccessToken)
	app.jsapiTicket.setGetTokenFunc(app.getJSAPITicket)
	app.jsapiTicketAgentConfig.setGetTokenFunc(app.getJSAPITicketAgentConfig)
	return &app
}

func (c *WorkwxApp) composeQyapiURL(path string, req interface{}) *url.URL {
	values := url.Values{}
	if valuer, ok := req.(urlValuer); ok {
		values = valuer.intoURLValues()
	}

	// TODO: refactor
	base, err := url.Parse(c.opts.QYAPIHost)
	if err != nil {
		// TODO: error_chain
		panic(fmt.Sprintf("qyapiHost invalid: host=%s err=%+v", c.opts.QYAPIHost, err))
	}

	base.Path = path
	base.RawQuery = values.Encode()

	return base
}

func (c *WorkwxApp) composeQyapiURLWithToken(path string, req interface{}, withAccessToken bool) *url.URL {
	url := c.composeQyapiURL(path, req)

	if !withAccessToken {
		return url
	}

	q := url.Query()
	q.Set("access_token", c.accessToken.getToken())
	url.RawQuery = q.Encode()

	return url
}

func (c *WorkwxApp) executeQiYeApiGet(path string, req urlValuer, respObj interface{}, withAccessToken bool) error {
	url := c.composeQyapiURLWithToken(path, req, withAccessToken)
	urlStr := url.String()

	resp, err := c.opts.HTTP.Get(urlStr)
	if err != nil {
		// TODO: error_chain
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(respObj)
	if err != nil {
		// TODO: error_chain
		return err
	}

	return nil
}

func (c *WorkwxApp) executeQiYePost(path string, req bodyer, respObj interface{}, withAccessToken bool) error {
	url := c.composeQyapiURLWithToken(path, req, withAccessToken)
	urlStr := url.String()

	body, err := req.intoBody()
	if err != nil {
		// TODO: error_chain
		return err
	}

	resp, err := c.opts.HTTP.Post(urlStr, "application/json", bytes.NewReader(body))
	if err != nil {
		// TODO: error_chain
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(respObj)
	if err != nil {
		// TODO: error_chain
		return err
	}
	return nil
}

func (c *WorkwxApp) executeCollyPost(path string, req bodyer, respObj *respMessageSend, withAccessToken bool) error {
	url := c.composeQyapiURLWithToken(path, req, withAccessToken)
	urlStr := url.String()

	body, err := req.intoBody()
	if err != nil {
		// TODO: error_chain
		return err
	}

	res := c.collyPost(urlStr, body)
	err = json.Unmarshal(res, respObj)
	if err != nil {
		logrus.Errorln(err)
		return err
	}

	if respObj.ErrCode != 0 {
		return fmt.Errorf("%s", respObj.ErrMsg)
	}
	return nil
}

func (c *WorkwxApp) executeQiYeApiMediaUpload(
	path string,
	req mediaUploader,
	respObj interface{},
	withAccessToken bool,
) error {
	url := c.composeQyapiURLWithToken(path, req, withAccessToken)
	urlStr := url.String()

	m := req.getMedia()

	// FIXME: use streaming upload to conserve memory!
	buf := bytes.Buffer{}
	mw := multipart.NewWriter(&buf)

	err := m.writeTo(mw)
	if err != nil {
		return err
	}

	err = mw.Close()
	if err != nil {
		return err
	}

	resp, err := c.opts.HTTP.Post(urlStr, mw.FormDataContentType(), &buf)
	if err != nil {
		// TODO: error_chain
		return err
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(respObj)
	if err != nil {
		// TODO: error_chain
		return err
	}

	return nil
}

func (c *WorkwxApp) collyGet(URL string) (body []byte) {
	u, err := url.Parse(URL)
	if err != nil {
		logrus.Errorln(err)
		return body
	}

	collyClient := colly.NewCollector()
	collyClient.SetRequestTimeout(100 * time.Second)
	collyClient.UserAgent = UserAgentForChrome

	start := time.Now()

	collyClient.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Host", u.Host)
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Accept", "*/*")
		r.Headers.Set("Origin", u.Host)
		r.Headers.Set("Referer", URL)
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("Accept-Language", "zh-CN, zh;q=0.9")
		r.Headers.Set("Content-Type", "application/json;charset=UTF-8")
	})

	collyClient.OnResponse(func(resp *colly.Response) {
		logrus.Infoln("Response from", u.Host)
		body = resp.Body
		return
	})
	collyClient.OnError(func(resp *colly.Response, err error) {
		logrus.Errorln(err)
	})

	if err = collyClient.Visit(URL); err != nil {
		logrus.Errorln(err)
	}
	eT := time.Since(start)
	logrus.Infoln("======> Send finished", u.Host, eT)
	return body
}

func (c *WorkwxApp) collyPost(URL string, data []byte) (body []byte) {
	u, err := url.Parse(URL)
	if err != nil {
		logrus.Errorln(err)
		return body
	}

	start := time.Now()
	defer func() {
		eT := time.Since(start)
		logrus.Infoln("===>", URL[len(u.Host)+1+len("http://"):], "|", eT)
	}()

	// collyClient := colly.NewCollector(colly.MaxDepth(1), colly.DetectCharset(), colly.Async(true), colly.AllowURLRevisit())
	collyClient := colly.NewCollector(colly.MaxDepth(1), colly.DetectCharset(), colly.AllowURLRevisit())
	collyClient.SetRequestTimeout(120 * time.Second)
	collyClient.UserAgent = GetUserAgent()

	collyClient.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Host", u.Host)
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Accept", "*/*")
		r.Headers.Set("Origin", u.Host)
		r.Headers.Set("Referer", URL)
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("Accept-Language", "zh-CN, zh;q=0.9")
		r.Headers.Set("Content-Type", "application/json;charset=UTF-8")
	})

	collyClient.OnResponse(func(resp *colly.Response) {
		body = resp.Body
		return
	})
	collyClient.OnError(func(resp *colly.Response, err error) {
		logrus.Errorln(err)
		logrus.Warnln(string(data))
	})

	err2 := collyClient.PostRaw(URL, data)
	if err2 != nil {
		logrus.Errorln(err2)
		return
	}
	collyClient.Wait()
	return body
}
