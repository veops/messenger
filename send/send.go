package send

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	"github.com/spf13/cast"

	"github.com/veops/messenger/global"
)

const (
	simpleText     = "text"
	simpleMarkdown = "markdown"
)

var (
	registered  = make(map[string]func(map[string]string) sender)
	msgCh       = make(chan *message, 10000)
	confCh      = make(chan struct{}, 1)
	name2sender = make(map[string]sender)
	rc          = resty.NewWithClient(&http.Client{})
)

type sender interface {
	send(*message) error
	getConf() map[string]string
}

type senderManager interface {
	sender
	getUIDByPhone(string) (string, error)
}

type message struct {
	Sender     string         `json:"sender" validate:"required" example:"myWechatBot"`
	MsgType    string         `json:"msgtype" validate:"required" example:"text"`
	Content    string         `json:"content" validate:"required" example:"this is a text content"`
	Title      string         `json:"title" validate:"optional" example:""`
	Tos        []string       `json:"tos" validate:"optional" example:""`
	Ccs        []string       `json:"ccs" validate:"optional" example:""`
	Extra      string         `json:"extra" validate:"optional" example:"{\"enable_duplicate_check\": 1,\"duplicate_check_interval\": 1800}"`
	Sync       bool           `json:"sync" validate:"optional" example:"true"`
	Simple     bool           `json:"simple" validate:"optional" example:"true"`
	Ats        []string       `json:"ats" validate:"optional" example:"xxx"`
	AtMobiles  []string       `json:"at_mobiles" validate:"optional" example:"133123456789"`
	ContentMap map[string]any `json:"-"`
	ExtraMap   map[string]any `json:"-"`
	Err        error          `json:"-"`
	Req        string         `json:"-"`
	Resp       string         `json:"-"`
	ReceivedAt int64          `json:"-"`
}

type getUIDByPhoneReq struct {
	Sender string `json:"sender" validate:"required" example:"myWechatBot"`
	Phone  string `json:"phone" validate:"required" example:"133123456789"`
}

func init() {
	rc.RetryCount = 3
	global.RegisterWatchCallbacks(func() {
		confCh <- struct{}{}
	})
}

func Start() error {
	for {
		select {
		case <-confCh:
			handleConfig()
		case msg := <-msgCh:
		PRIORITY:
			for {
				select {
				case <-confCh:
					fmt.Println(3)
					handleConfig()
				default:
					break PRIORITY
				}
			}
			go handleMessage(msg)
		}
	}
}

// PushMessage
//
//	@Tags			send
//	@Description	send a new message
//	@Description	https://github.com/veops/messenger?tab=readme-ov-file#发送消息
//	@Accept			json
//	@Produce		json
//	@Param			body	body		message				true	" "
//	@Success		200		{object}	map[string]string	"a map with msg info, eg. {msg:ok}"
//	@Router			/v1/message [POST]
func PushMessage(ctx *gin.Context) {
	m := &message{}
	if err := ctx.ShouldBindBodyWith(&m, binding.JSON); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	m.ReceivedAt = time.Now().Unix()
	m.Ats = lo.Uniq(m.Ats)
	m.AtMobiles = lo.Uniq(m.AtMobiles)

	if s, ok := name2sender[m.Sender]; ok && s != nil && s.getConf()["type"] != "email" && !m.Simple {
		if m.Content != "" {
			if err := json.Unmarshal([]byte(cast.ToString(m.Content)), &m.ContentMap); err != nil {
				ctx.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}
	}
	if m.Extra != "" {
		if err := json.Unmarshal([]byte(cast.ToString(m.Extra)), &m.ExtraMap); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	if m.Sync {
		if err := handleMessage(m); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			log.Println(err)
		}
		return
	}

	msgCh <- m
}

// GetUIDByPhone
//
//	@Tags			send
//	@Description	get user's uid by he or she's phone number
//	@Description	https://github.com/veops/messenger?tab=readme-ov-file#查询用户ID
//	@Accept			json
//	@Produce		json
//	@Param			body	body		getUIDByPhoneReq	true	" "
//	@Success		200		{object}	map[string]string	"a map with phone as key and uid as value"
//	@Router			/v1/uid/getbyphone [POST]
func GetUIDByPhone(ctx *gin.Context) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		if err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err)
		}
	}()
	r := &getUIDByPhoneReq{}
	if err := ctx.ShouldBindBodyWith(&r, binding.JSON); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	s, ok := name2sender[r.Sender]
	if !ok || s == nil {
		err = fmt.Errorf("cannot find sender with name %s", r.Sender)
		return
	}
	sm, ok := s.(senderManager)
	if !ok || sm == nil {
		err = fmt.Errorf("sender with name %s and type %s does not support to query uid by phone", r.Sender, s.getConf()["type"])
		return
	}
	uid, err := sm.getUIDByPhone(r.Phone)
	if err != nil {
		return
	}
	ctx.JSON(http.StatusOK, map[string]string{"uid": uid})
}

func handleErr(info string, e error, resp *resty.Response, isOk func(dt map[string]any) bool) error {
	if e != nil {
		return e
	}

	dt := make(map[string]any)
	_ = json.Unmarshal(resp.Body(), &dt)
	if resp.StatusCode() != 200 || !isOk(dt) {
		return fmt.Errorf("%s httpcode=%v resp=%s", info, resp.StatusCode(), global.RenderPretty(dt))
	}

	return nil
}

func handleConfig() {
	confs, err := global.GetSenders()
	if err != nil {
		log.Println(err)
		return
	}

	valid := make(map[string]struct{})
	for _, conf := range confs {
		name := conf["name"]
		valid[name] = struct{}{}
		if s, ok := name2sender[name]; !ok || s == nil || !reflect.DeepEqual(conf, s.getConf()) {
			f, ok := registered[conf["type"]]
			if !ok || f == nil {
				continue
			}
			name2sender[name] = f(conf)
		}
	}

	for k, v := range name2sender {
		if _, ok := valid[k]; !ok || v == nil {
			delete(name2sender, k)
		}
	}
}

func handleMessage(msg *message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		if err != nil && !msg.Sync {
			log.Println(err)
		}
		if msg.Err == nil {
			msg.Err = err
		}
		AddHistory(msg)
	}()

	s, ok := name2sender[msg.Sender]
	if !ok || s == nil {
		err = fmt.Errorf("cannot find sender with name %s", msg.Sender)
		return
	}

	if err = s.send(msg); err != nil {
		err = fmt.Errorf("send failed %w", err)
	}

	return
}
