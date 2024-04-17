package biliapi

import (
	"encoding/json"
	"errors"
	"net/http"

	f "github.com/qydysky/bili_danmu/F"
	cmp "github.com/qydysky/part/component2"
	part "github.com/qydysky/part/pool"
	reqf "github.com/qydysky/part/reqf"
)

const pkgId = "github.com/qydysky/bili_danmu/F"

func init() {
	if e := cmp.Register[f.BiliApi](pkgId, &biliApi{}); e != nil {
		panic(e)
	}
}

type biliApi struct {
	proxy string
}

// LoginQrPoll implements F.BiliApi.
func (t *biliApi) LoginQrPoll(pool *part.Buf[reqf.Req], QrcodeKey string) (err error, cookies []*http.Cookie) {
	r := pool.Get()
	defer pool.Put(r)
	if e := r.Reqf(reqf.Rval{
		Url:     `https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=` + QrcodeKey + `&source=main-fe-header`,
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
		Retry:   2,
	}); e != nil {
		err = e
		return
	}

	var res struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			URL          string `json:"url"`
			RefreshToken string `json:"refresh_token"`
			Timestamp    int    `json:"timestamp"`
			Code         int    `json:"code"`
			Message      string `json:"message"`
		} `json:"data"`
	}

	if e := json.Unmarshal(r.Respon, &res); e != nil {
		err = e
		return
	}

	if res.Code != 0 {
		err = errors.New(`code != 0`)
		return
	} else if res.Data.Code == 0 {
		cookies = r.Response.Cookies()
	}
	return
}

func (t *biliApi) Proxy(proxy string) f.BiliApi {
	t.proxy = proxy
	return t
}

func (t *biliApi) LoginQrCode(pool *part.Buf[reqf.Req]) (err error, imgUrl string, QrcodeKey string) {
	r := pool.Get()
	defer pool.Put(r)
	if e := r.Reqf(reqf.Rval{
		Url:     `https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header`,
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
		Retry:   2,
	}); e != nil {
		err = e
		return
	}

	var res struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			URL       string `json:"url"`
			QrcodeKey string `json:"qrcode_key"`
		} `json:"data"`
	}

	if e := json.Unmarshal(r.Respon, &res); e != nil {
		err = e
		return
	}
	if res.Code != 0 {
		err = errors.New(`code != 0`)
		return
	}

	if res.Data.URL == `` {
		err = errors.New(`Data.URL == ""`)
		return
	} else {
		imgUrl = res.Data.URL
	}
	if res.Data.QrcodeKey == `` {
		err = errors.New(`Data.QrcodeKey == ""`)
		return
	} else {
		QrcodeKey = res.Data.QrcodeKey
	}
	return
}
