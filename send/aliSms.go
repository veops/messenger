package send

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cast"
)

const (
	aliSmsUrl = "http://dysmsapi.aliyuncs.com"
)

func init() {
	registered["aliSms"] = func(conf map[string]string) sender {
		return &aliSms{conf: conf}
	}
}

type aliSms struct {
	conf map[string]string
}

// send ali sms
//
//	https://help.aliyun.com/zh/sms/developer-reference/api-dysmsapi-2017-05-25-sendsms
func (a *aliSms) send(msg *message) error {
	body := map[string]string{
		"PhoneNumbers":  strings.Join(msg.Tos, ","),
		"SignName":      a.conf["signName"],
		"TemplateCode":  a.conf["templateCode"],
		"TemplateParam": msg.Content,
	}
	req := rc.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetQueryParams(map[string]string{
			"Action":           "SendSms",
			"Version":          "2017-05-25",
			"Format":           "JSON",
			"AccessKeyId":      a.conf["accessKey"],
			"SignatureNonce":   cast.ToString(rand.Int63()),
			"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			"SignatureMethod":  "HMAC-SHA1",
			"SignatureVersion": "1.0",
			"AcceptLanguage":   "zh-CN",
		}).
		SetFormData(body)
	ks := lo.Keys(body)
	for k, _ := range req.QueryParam {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	encodeParams := make([]string, 0)
	for _, k := range ks {
		v, ok := body[k]
		if !ok {
			v = req.QueryParam.Get(k)
		}
		encodeParams = append(encodeParams, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
	}
	CanonicalizedQueryString := strings.Join(encodeParams, "&")
	stringToSign := fmt.Sprintf("%s&%s&%s",
		"POST",
		url.QueryEscape("/"),
		url.QueryEscape(CanonicalizedQueryString),
	)
	hashed := hmac.New(sha1.New, []byte(a.conf["accessSecret"]+"&"))
	hashed.Write([]byte(stringToSign))

	signature := base64.StdEncoding.EncodeToString(hashed.Sum(nil))
	req.SetQueryParam("Signature", signature)

	resp, err := req.Post(aliSmsUrl)

	return handleErr("send to ali sms failed", err, resp, func(dt map[string]any) bool { return dt["Code"] == "OK" })
}

func (a *aliSms) getConf() map[string]string {
	return a.conf
}
