package biliApi

import (
	"errors"
	"testing"

	cmp "github.com/qydysky/part/component2"
	pool "github.com/qydysky/part/pool"
	reqf "github.com/qydysky/part/reqf"
)

var api biliApiInter

func init() {
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
	api = cmp.Get(id, cmp.PreFuncCu[biliApiInter]{
		Initf: func(bai biliApiInter) biliApiInter {
			bai.SetReqPool(reqPool)
			return bai
		},
	})
}

func TestGetInfoByRoom(t *testing.T) {
	if err, _ := api.GetInfoByRoom(213); err != nil {
		t.Fatal(err)
	}
}

func TestMain(t *testing.T) {
	if err, _, QrcodeKey := api.LoginQrCode(); err != nil {
		t.Fatal(err)
	} else if err, _ := api.LoginQrPoll(QrcodeKey); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.GetRoomBaseInfo(213); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.LiveHtml(92613); err != nil {
		t.Fatal(err)
	}

	if err := api.GetOtherCookies(); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.GetNav(); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.SearchUP("C酱"); err != nil {
		t.Fatal(err)
	}

	if err := api.IsConnected(); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.GetHisDanmu(213); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.GetFollowing(); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}

	if err, _ := api.GetOnlineGoldRank(13046, 92613); err != nil {
		t.Fatal(err)
	}

	if err := api.RoomEntryAction(92613); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}

	if err, _ := api.GetHisStream(); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}

	if err, _ := api.Silver2coin(); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}

	if err, _ := api.GetWalletRule(); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}

	if err, _ := api.GetWalletStatus(); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}

	if err, _ := api.GetBagList(213); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}

	if err := api.GetLiveBuvid(213); err != nil {
		t.Fatal(err)
	}

	if err, _ := api.DoSign(); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}
	if err, _ := api.GetWebGetSignInfo(); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}
	if err := api.SetFansMedal(0); err.Error() != `405 Method Not Allowed` {
		t.Fatal(err)
	}
	if err, _ := api.GetFansMedal(213, 0); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}
	if err, _ := api.GetWearedMedal(29183321, 92613); !errors.Is(err, ErrNeedLogin) {
		t.Fatal(err)
	}
	if err, _ := api.GetGuardNum(13046, 92613); err != nil {
		t.Fatal(err)
	}
	if err, _ := api.GetPopularAnchorRank(0, 13046, 92613); err != nil {
		t.Fatal(err)
	}
}
