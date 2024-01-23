package send

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

func init() {
	registered["wechatBot"] = func(conf map[string]string) sender {
		return &wechatBot{conf: conf}
	}
}

type wechatBot struct {
	conf map[string]string
}

// send wechat bot message
//
//	https://developer.work.weixin.qq.com/document/path/99110
func (w *wechatBot) send(msg *message) error {
	if msg.Simple {
		switch msg.MsgType {
		case simpleText, simpleMarkdown:
			msg.ContentMap = map[string]any{
				"content": msg.Content,
			}
		default:
			return fmt.Errorf("sender type %s does not support simple type %s", w.conf["type"], msg.MsgType)
		}
	}

	if len(msg.Ats) > 0 {
		switch msg.MsgType {
		case simpleText:
			msg.ContentMap["mentioned_list"] = msg.Ats
		case simpleMarkdown:
			msg.ContentMap["content"] = fmt.Sprintf("%v \n %s",
				msg.ContentMap["content"],
				strings.Join(lo.Map(msg.Ats, func(s string, _ int) string { return fmt.Sprintf("<@%s>", s) }), " "))
		}
	}
	if len(msg.AtMobiles) > 0 {
		switch msg.MsgType {
		case simpleText:
			msg.ContentMap["mentioned_mobile_list"] = msg.AtMobiles
		}
	}

	resp, err := rc.SetPreRequestHook(RecordHttpReq(msg)).R().
		SetBody(lo.Assign(msg.ExtraMap, map[string]any{
			"msgtype":   msg.MsgType,
			msg.MsgType: msg.ContentMap,
		})).
		Post(w.conf["url"])

	RecordResp(msg, err, resp)

	return handleErr("wechat bot send failed", err, resp, func(dt map[string]any) bool { return dt["errcode"] == 0.0 })
}

func (w *wechatBot) getConf() map[string]string {
	return w.conf
}
