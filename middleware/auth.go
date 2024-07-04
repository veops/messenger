package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

func Auth(confs []map[string]string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(confs) <= 0 {
			ctx.Next()
			return
		}
		for _, conf := range confs {
			t := cast.ToString(conf["type"])
			if type2auth[t] == nil || !type2auth[t](conf, ctx) {
				continue
			}
			ctx.Next()
			return
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
	}
}

var (
	type2auth = map[string]func(map[string]string, *gin.Context) bool{
		"ip":    authByIP,
		"token": authByToken,
		"sign":  authBySign,
	}
)

func authByIP(conf map[string]string, ctx *gin.Context) bool {
	ip := ctx.ClientIP()
	ps := strings.Split(conf["pattern"], ",")
	for _, p := range ps {
		ss1 := strings.Split(p, ".")
		ss2 := strings.Split(ip, ".")
		if len(ss1) != len(ss2) {
			continue
		}
		b := true
		for i := 0; i < len(ss1) && b; i++ {
			if ss1[i] != "*" && ss1[i] != ss2[i] {
				b = false
			}
		}
		if b {
			return true
		}
	}
	return false
}

func authByToken(conf map[string]string, ctx *gin.Context) bool {
	return conf["token"] != "" && conf["token"] == ctx.GetHeader("X-Token")
}

func authBySign(conf map[string]string, ctx *gin.Context) bool {
	ts := cast.ToInt64(ctx.GetHeader("X-TS"))
	now := time.Now()
	t := time.Unix(ts, 0)
	if ts == 0 || t.Add(time.Second*60).Before(now) || t.Add(-time.Second*60).After(now) {
		return false
	}

	body := make(map[string]string)
	if ctx.ShouldBindBodyWith(&body, binding.JSON) != nil {
		return false
	}

	body["nonce"] = ctx.GetHeader("X-Nonce")
	body["ts"] = ctx.GetHeader("X-TS")

	keys := lo.Keys(body)
	sort.Strings(keys)
	kvStr := strings.Join(lo.Map(keys, func(k string, _ int) string { return fmt.Sprintf("%s%s", k, body[k]) }), "")

	mac := hmac.New(sha256.New, []byte(conf["secret"]))
	_, _ = mac.Write([]byte(kvStr))
	return ctx.GetHeader("X-Sign") == base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
