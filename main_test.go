package biliApi

import (
	"testing"

	cmp "github.com/qydysky/part/component2"
	pool "github.com/qydysky/part/pool"
	reqf "github.com/qydysky/part/reqf"
)

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
	var api = cmp.Get(id, func(bai biliApiInter) biliApiInter {
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
