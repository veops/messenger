package send

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/moul/http2curl"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	"gopkg.in/gomail.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func init() {
	var err error
	db, err = gorm.Open(sqlite.Open("history.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("init sqlite failed, err=%v", err)
	}
	err = db.AutoMigrate(History{})
	if err != nil {
		log.Fatalf("migrate failed, err=%v", err)
	}
}

type History struct {
	Id         int    `gorm:"column:id"`
	Message    string `gorm:"column:message"`
	Err        string `gorm:"column:err"`
	Req        string `gorm:"column:req"`
	Resp       string `gorm:"column:resp"`
	Status     bool   `gorm:"column:status"`
	ReceivedAt int64  `gorm:"column:received_at"`
	CreatedAt  int64  `gorm:"column:created_at"`
}

func (History) TableName() string {
	return "history"
}

func AddHistory(msg *message) {
	bs, _ := json.Marshal(msg)
	err := ""
	if msg.Err != nil {
		err = msg.Err.Error()
	}
	if err := db.Create(&History{
		Message:    string(bs),
		Err:        err,
		Req:        msg.Req,
		Resp:       msg.Resp,
		Status:     msg.Err == nil,
		ReceivedAt: msg.ReceivedAt,
	}).Error; err != nil {
		log.Printf("add history failed,err=%v", err)
	}
}

// QueryHistory
//
//	@Tags			send
//	@Description	query message history
//	@Param			page_index	query		int		true	"page_index"
//	@Param			page_size	query		int		true	"page_size"
//	@Param			start		query		int		false	"start time"
//	@Param			end			query		int		false	"end time"
//	@Param			status		query		string	false	"false failed, true sent successfully"
//	@Param			sender		query		string	false	"sender name"
//	@Param			content		query		string	false	"content"
//	@Success		200			{object}	map[string]any
//	@Router			/v1/history [GET]
func QueryHistory(ctx *gin.Context) {
	pageIndex, pageSize := cast.ToInt(ctx.Query("page_index")), cast.ToInt(ctx.Query("page_size"))
	q := db.Model(&History{}).Offset((pageIndex - 1) * pageSize).Limit(pageSize)
	if v, ok := ctx.GetQuery("start"); ok {
		q = q.Where("received_at >= ?", v)
	}
	if v, ok := ctx.GetQuery("end"); ok {
		q = q.Where("received_at <= ?", v)
	}
	if v, ok := ctx.GetQuery("status"); ok {
		q = q.Where("status = ?", cast.ToBool(v))
	}
	for _, k := range []string{"sender", "content"} {
		if v, ok := ctx.GetQuery(k); ok {
			q = q.Where(fmt.Sprintf("JSON_EXTRACT(`message`,'$.%s') LIKE ?", k), fmt.Sprintf("%%%s%%", v))
		}
	}
	count := int64(0)
	histories := make([]*History, 0)
	cfg := &gorm.Session{}
	eg := errgroup.Group{}
	eg.Go(func() error {
		return q.Session(cfg).Count(&count).Error
	})
	eg.Go(func() error {
		return q.Session(cfg).Find(&histories).Error
	})

	if err := eg.Wait(); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"count": count,
		"list":  histories,
	})
}

func RecordHttpReq(msg *message) resty.PreRequestHook {
	return func(c *resty.Client, r *http.Request) error {
		curl, _ := http2curl.GetCurlCommand(r)
		msg.Req = curl.String()
		return nil
	}
}

func RecordEmailReq(msg *message, m *gomail.Message) {
	buf := &bytes.Buffer{}
	m.WriteTo(buf)
	msg.Req = buf.String()
}

func RecordResp(msg *message, err error, resp *resty.Response) {
	if err != nil {
		msg.Err = err
	}
	if resp == nil {
		return
	}
	m := map[string]any{
		"httpCode": resp.StatusCode(),
		"body":     resp.String(),
	}
	bs, _ := json.Marshal(m)
	msg.Resp = string(bs)
}
