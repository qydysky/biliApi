package biliApi

import (
	"net/http"
	"testing"
	"time"

	cmp "github.com/qydysky/part/component2"
	pool "github.com/qydysky/part/pool"
	reqf "github.com/qydysky/part/reqf"
)

type biliApiInter1 interface {
	SetReqPool(pool *pool.Buf[reqf.Req])
	SetProxy(proxy string)
	SetCookies(cookies []*http.Cookie)

	LoginQrCode() (err error, imgUrl string, QrcodeKey string)
	LoginQrPoll(QrcodeKey string) (err error, cookies []*http.Cookie)
	GetRoomBaseInfo(Roomid int) (err error, res struct {
		UpUid         int
		Uname         string
		ParentAreaID  int
		AreaID        int
		Title         string
		LiveStartTime time.Time
		Liveing       bool
		RoomID        int
	})
	GetInfoByRoom(Roomid int) (err error, res struct {
		UpUid         int
		Uname         string
		ParentAreaID  int
		AreaID        int
		Title         string
		LiveStartTime time.Time
		Liveing       bool
		RoomID        int
		GuardNum      int
		Note          string
		Locked        bool
	})
}

func TestMain(t *testing.T) {
	var reqPool = pool.New(
		pool.PoolFunc[reqf.Req]{
			New: func() *reqf.Req {
				return reqf.New()
			},
			InUse: func(r *reqf.Req) bool {
				return r.IsLive()
			},
			Reuse: func(r *reqf.Req) *reqf.Req {
				return r
			},
			Pool: func(r *reqf.Req) *reqf.Req {
				return r
			},
		},
		100,
	)
	var api = cmp.Get(id, func(bai biliApiInter1) biliApiInter1 {
		bai.SetReqPool(reqPool)
		return bai
	})

	if err, _, QrcodeKey := api.LoginQrCode(); err != nil {
		t.Fatal(err)
	} else if err, _ := api.LoginQrPoll(QrcodeKey); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.GetRoomBaseInfo(213); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.GetInfoByRoom(213); err != nil {
		t.Fatal(err)
	}
}
