package biliApi

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	cmp "github.com/qydysky/part/component2"
	pool "github.com/qydysky/part/pool"
	reqf "github.com/qydysky/part/reqf"
)

const id = "github.com/qydysky/bili_danmu/F.biliApi"
const UA = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.3`

func init() {
	if e := cmp.Register[biliApiInter](id, &biliApi{}); e != nil {
		panic(e)
	}
}

type biliApi struct {
	proxy   string
	pool    *pool.Buf[reqf.Req]
	cookies []*http.Cookie
}

// DoSign implements biliApiInter.
func (t *biliApi) DoSign() (err error, HadSignDays int) {
	req := t.pool.Get()
	defer t.pool.Put(req)
	err = req.Reqf(reqf.Rval{
		Url: `https://api.live.bilibili.com/xlive/web-ucenter/v1/sign/DoSign`,
		Header: map[string]string{
			`Host`:            `api.live.bilibili.com`,
			`User-Agent`:      UA,
			`Accept`:          `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`:          `https://live.bilibili.com`,
			`Connection`:      `keep-alive`,
			`Pragma`:          `no-cache`,
			`Cache-Control`:   `no-cache`,
			`Referer`:         "https://live.bilibili.com/all",
			`Cookie`:          reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 3 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			HadSignDays int `json:"hadSignDays"`
		} `json:"data"`
	}
	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	HadSignDays = j.Data.HadSignDays
	return
}

// GetWebGetSignInfo implements biliApiInter.
func (t *biliApi) GetWebGetSignInfo() (err error, Status int) {

	req := t.pool.Get()
	defer t.pool.Put(req)
	err = req.Reqf(reqf.Rval{
		Url: `https://api.live.bilibili.com/xlive/web-ucenter/v1/sign/WebGetSignInfo`,
		Header: map[string]string{
			`Host`:            `api.live.bilibili.com`,
			`User-Agent`:      UA,
			`Accept`:          `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`:          `https://live.bilibili.com`,
			`Connection`:      `keep-alive`,
			`Pragma`:          `no-cache`,
			`Cache-Control`:   `no-cache`,
			`Referer`:         "https://live.bilibili.com/all",
			`Cookie`:          reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 3 * 1000,
		Retry:   2,
	})

	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status int `json:"status"`
		} `json:"data"`
	}

	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}
	Status = j.Data.Status
	return
}

// GetCookies implements biliApiInter.
func (t *biliApi) GetCookie(name string) (error, string) {
	for i := 0; i < len(t.cookies); i++ {
		if t.cookies[i].Name == name {
			return nil, t.cookies[i].Value
		}
	}
	return errors.New("not found"), ""
}

// SetFansMedal implements biliApiInter.
func (t *biliApi) SetFansMedal(medalId int) (err error) {
	post_url := `https://api.live.bilibili.com/xlive/web-room/v1/fansMedal/take_off` //无牌，不佩戴牌子
	post_str := ""

	if medalId != 0 {
		e, csrf := t.GetCookie(`bili_jct`)
		if e != nil {
			return e
		}
		post_url = `https://api.live.bilibili.com/xlive/web-room/v1/fansMedal/wear`
		post_str = fmt.Sprintf("medal_id=%d&csrf_token=%s&csrf=%s", medalId, csrf, csrf)
	}

	r := t.pool.Get()
	defer t.pool.Put(r)
	err = r.Reqf(reqf.Rval{
		Url:     post_url,
		PostStr: post_str,
		Header: map[string]string{
			`Cookie`:       reqf.Cookies_List_2_String(t.cookies),
			`Content-Type`: `application/x-www-form-urlencoded; charset=UTF-8`,
			`Referer`:      `https://passport.bilibili.com/login`,
		},
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
	}

	err = json.Unmarshal(r.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	return
}

// GetFansMedal implements biliApiInter.
func (t *biliApi) GetFansMedal(RoomID, TargetID int) (err error, res []struct {
	TargetID  int
	IsLighted int
	MedalID   int
	RoomID    int
}) {
	//获取牌子列表
	r := t.pool.Get()
	defer t.pool.Put(r)

	for pageNum := 1; true; pageNum += 1 {
		url := fmt.Sprintf("https://api.live.bilibili.com/xlive/app-ucenter/v1/fansMedal/panel?page=%d&page_size=10", pageNum)
		if RoomID != 0 {
			url += fmt.Sprintf("&room_id=%d", RoomID)
		}
		if TargetID != 0 {
			url += fmt.Sprintf("&target_id=%d", TargetID)
		}

		err = r.Reqf(reqf.Rval{
			Url: url,
			Header: map[string]string{
				`Cookie`: reqf.Cookies_List_2_String(t.cookies),
			},
			Proxy:   t.proxy,
			Timeout: 10 * 1000,
			Retry:   2,
		})
		if err != nil {
			return
		}

		var j struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			TTL     int    `json:"ttl"`
			Data    struct {
				List []struct {
					Medal struct {
						TargetID  int `json:"target_id"`
						MedalID   int `json:"medal_id"`
						IsLighted int `json:"is_lighted"`
					} `json:"medal"`
					AnchorInfo struct {
						NickName string `json:"nick_name"`
					} `json:"anchor_info"`
					RoomInfo struct {
						RoomID int `json:"room_id"`
					} `json:"room_info"`
				} `json:"list"`
				SpecialList []struct {
					Medal struct {
						TargetID  int `json:"target_id"`
						MedalID   int `json:"medal_id"`
						IsLighted int `json:"is_lighted"`
					} `json:"medal"`
					AnchorInfo struct {
						NickName string `json:"nick_name"`
					} `json:"anchor_info"`
					RoomInfo struct {
						RoomID int `json:"room_id"`
					} `json:"room_info"`
				} `json:"special_list"`
				PageInfo struct {
					CurrentPage int `json:"current_page"`
					TotalPage   int `json:"total_page"`
				} `json:"page_info"`
			} `json:"data"`
		}

		err = json.Unmarshal(r.Respon, &j)
		if err != nil {
			return
		} else if j.Code != 0 {
			err = errors.New(j.Message)
			return
		}

		for i := 0; i < len(j.Data.SpecialList); i++ {
			li := j.Data.SpecialList[i]
			res = append(res, struct {
				TargetID  int
				IsLighted int
				MedalID   int
				RoomID    int
			}{
				TargetID:  li.Medal.TargetID,
				IsLighted: li.Medal.IsLighted,
				MedalID:   li.Medal.MedalID,
				RoomID:    li.RoomInfo.RoomID,
			})
			if RoomID != 0 && li.RoomInfo.RoomID == RoomID {
				return
			}
			if TargetID != 0 && li.Medal.TargetID == TargetID {
				return
			}
		}

		for i := 0; i < len(j.Data.List); i++ {
			li := j.Data.List[i]
			res = append(res, struct {
				TargetID  int
				IsLighted int
				MedalID   int
				RoomID    int
			}{
				TargetID:  li.Medal.TargetID,
				IsLighted: li.Medal.IsLighted,
				MedalID:   li.Medal.MedalID,
				RoomID:    li.RoomInfo.RoomID,
			})
			if RoomID != 0 && li.RoomInfo.RoomID == RoomID {
				return
			}
			if TargetID != 0 && li.Medal.TargetID == TargetID {
				return
			}
		}

		if j.Data.PageInfo.CurrentPage == j.Data.PageInfo.TotalPage {
			break
		}

		time.Sleep(time.Second)
	}

	return
}

// GetWearedMedal implements biliApiInter.
func (t *biliApi) GetWearedMedal() (err error, res struct {
	TodayIntimacy int
	RoomID        int
	TargetID      int
}) {
	r := t.pool.Get()
	defer t.pool.Put(r)
	err = r.Reqf(reqf.Rval{
		Url: `https://api.live.bilibili.com/live_user/v1/UserInfo/get_weared_medal`,
		Header: map[string]string{
			`Cookie`: reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	}
	var jd struct {
		TodayIntimacy int `json:"today_intimacy"`
		TargetID      int `json:"target_id"`
		Roominfo      struct {
			RoomID int `json:"room_id"`
		} `json:"roominfo"`
	}

	err = json.Unmarshal(r.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	switch j.Data.(type) {
	case any:
		return
	default:
		if data, e := json.Marshal(j.Data); e != nil {
			err = e
			return
		} else if e = json.Unmarshal(data, &jd); e != nil {
			err = e
			return
		} else {
			res.TodayIntimacy = jd.TodayIntimacy
			res.TargetID = jd.TargetID
			res.RoomID = jd.Roominfo.RoomID
			return
		}
	}
}

// Wbi implements biliApiInter.
func (t *biliApi) Wbi(query string, WbiImg struct {
	ImgURL string
	SubURL string
}) (err error, queryEnc string) {
	if query != "" {
		wrid, wts := getWridWts(query, WbiImg.ImgURL, WbiImg.SubURL)
		queryEnc = query + "&w_rid=" + wrid + "&wts=" + wts
	}
	return
}

// GetNav implements biliApiInter.
func (t *biliApi) GetNav() (err error, res struct {
	IsLogin bool
	WbiImg  struct {
		ImgURL string
		SubURL string
	}
}) {
	req := t.pool.Get()
	defer t.pool.Put(req)
	err = req.Reqf(reqf.Rval{
		Url: `https://api.bilibili.com/x/web-interface/nav`,
		Header: map[string]string{
			`Host`:            `api.bilibili.com`,
			`User-Agent`:      UA,
			`Accept`:          `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`:          `https://t.bilibili.com`,
			`Connection`:      `keep-alive`,
			`Pragma`:          `no-cache`,
			`Cache-Control`:   `no-cache`,
			`Referer`:         `https://t.bilibili.com/pages/nav/index_new`,
			`Cookie`:          reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 3 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			IsLogin bool `json:"isLogin"`
			WbiImg  struct {
				ImgURL string `json:"img_url"`
				SubURL string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}

	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else {
		res.IsLogin = j.Data.IsLogin
		res.WbiImg.ImgURL = j.Data.WbiImg.ImgURL
		res.WbiImg.SubURL = j.Data.WbiImg.SubURL
	}

	return
}

// GetGuardNum implements biliApiInter.
func (t *biliApi) GetGuardNum(upUid int, roomid int) (err error, GuardNum int) {
	req := t.pool.Get()
	defer t.pool.Put(req)

	err = req.Reqf(reqf.Rval{
		Url: fmt.Sprintf(`https://api.live.bilibili.com/xlive/app-room/v2/guardTab/topList?roomid=%d&page=1&ruid=%d&page_size=29`, roomid, upUid),
		Header: map[string]string{
			`Host`:            `api.live.bilibili.com`,
			`User-Agent`:      UA,
			`Accept`:          `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`:          `https://live.bilibili.com`,
			`Connection`:      `keep-alive`,
			`Pragma`:          `no-cache`,
			`Cache-Control`:   `no-cache`,
			`Referer`:         fmt.Sprintf("https://live.bilibili.com/%d", roomid),
			`Cookie`:          reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 3 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			Info struct {
				Num int `json:"num"`
			} `json:"info"`
		} `json:"data"`
	}

	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	//获取舰长数
	GuardNum = j.Data.Info.Num

	return
}

// GetPopularAnchorRank implements biliApiInter.
func (t *biliApi) GetPopularAnchorRank(uid int, upUid int, roomid int) (err error, note string) {
	req := t.pool.Get()
	defer t.pool.Put(req)

	err = req.Reqf(reqf.Rval{
		Url: fmt.Sprintf(`https://api.live.bilibili.com/xlive/general-interface/v1/rank/getPopularAnchorRank?uid=%d&ruid=%d&clientType=2`, uid, upUid),
		Header: map[string]string{
			`Host`:            `api.live.bilibili.com`,
			`User-Agent`:      UA,
			`Accept`:          `application/json, text/plain, */*`,
			`Accept-Language`: `zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2`,
			`Accept-Encoding`: `gzip, deflate, br`,
			`Origin`:          `https://live.bilibili.com`,
			`Connection`:      `keep-alive`,
			`Pragma`:          `no-cache`,
			`Cache-Control`:   `no-cache`,
			`Referer`:         fmt.Sprintf("https://live.bilibili.com/%d", roomid),
			`Cookie`:          reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 3 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			Anchor struct {
				Rank int `json:"rank"`
			} `json:"anchor"`
		} `json:"data"`
	}

	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	//获取排名
	note = "人气榜 "
	if j.Data.Anchor.Rank == 0 {
		note += "100+"
	} else {
		note += strconv.Itoa(j.Data.Anchor.Rank)
	}

	return
}

// getDanmuMedalAnchorInfo implements biliApiInter.
func (t *biliApi) GetDanmuMedalAnchorInfo(Uid string, Roomid int) (err error, rface string) {
	req := t.pool.Get()
	defer t.pool.Put(req)

	err = req.Reqf(reqf.Rval{
		Url: "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuMedalAnchorInfo?ruid=" + Uid,
		Header: map[string]string{
			`Referer`: fmt.Sprintf("https://live.bilibili.com/%d", Roomid),
			`Cookie`:  reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Rface string `json:"rface"`
		} `json:"data"`
	}

	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	rface = j.Data.Rface + `@58w_58h`

	return
}

// GetDanmuInfo implements biliApiInter.
func (t *biliApi) GetDanmuInfo(Roomid int) (err error, res struct {
	Token string
	WSURL []string
}) {
	req := t.pool.Get()
	defer t.pool.Put(req)
	err = req.Reqf(reqf.Rval{
		Url: fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?type=0&id=%d", Roomid),
		Header: map[string]string{
			`Referer`: fmt.Sprintf("https://live.bilibili.com/%d", Roomid),
			`Cookie`:  reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			Group            string  `json:"group"`
			BusinessID       int     `json:"business_id"`
			RefreshRowFactor float64 `json:"refresh_row_factor"`
			RefreshRate      int     `json:"refresh_rate"`
			MaxDelay         int     `json:"max_delay"`
			Token            string  `json:"token"`
			HostList         []struct {
				Host    string `json:"host"`
				Port    int    `json:"port"`
				WssPort int    `json:"wss_port"`
				WsPort  int    `json:"ws_port"`
			} `json:"host_list"`
		} `json:"data"`
	}

	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	//弹幕钥
	res.Token = j.Data.Token
	//弹幕链接
	var tmp []string
	for _, v := range j.Data.HostList {
		if v.WssPort != 443 {
			tmp = append(tmp, "wss://"+v.Host+":"+strconv.Itoa(v.WssPort)+"/sub")
		} else {
			tmp = append(tmp, "wss://"+v.Host+"/sub")
		}
	}
	res.WSURL = tmp
	return
}

// GetRoomPlayInfo implements biliApiInter.
func (t *biliApi) GetRoomPlayInfo(Roomid int, Qn int) (err error, res struct {
	UpUid         int
	RoomID        int
	LiveStartTime time.Time
	Liveing       bool
	Streams       []struct {
		ProtocolName string
		Format       []struct {
			FormatName string
			Codec      []struct {
				CodecName string
				CurrentQn int
				AcceptQn  []int
				BaseURL   string
				URLInfo   []struct {
					Host      string
					Extra     string
					StreamTTL int
				}
				HdrQn     any
				DolbyType int
				AttrName  string
			}
		}
	}
}) {
	req := t.pool.Get()
	defer t.pool.Put(req)
	err = req.Reqf(reqf.Rval{
		Url: fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo?protocol=0,1&format=0,1,2&codec=0,1,2&qn=%d&platform=web&ptype=8&dolby=5&panorama=1&room_id=%d", Qn, Roomid),
		Header: map[string]string{
			`Referer`: fmt.Sprintf("https://live.bilibili.com/%d", Roomid),
			`Cookie`:  reqf.Cookies_List_2_String(t.cookies),
		},
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	var j struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			RoomID          int   `json:"room_id"`
			ShortID         int   `json:"short_id"`
			UID             int   `json:"uid"`
			IsHidden        bool  `json:"is_hidden"`
			IsLocked        bool  `json:"is_locked"`
			IsPortrait      bool  `json:"is_portrait"`
			LiveStatus      int   `json:"live_status"`
			HiddenTill      int   `json:"hidden_till"`
			LockTill        int   `json:"lock_till"`
			Encrypted       bool  `json:"encrypted"`
			PwdVerified     bool  `json:"pwd_verified"`
			LiveTime        int   `json:"live_time"`
			RoomShield      int   `json:"room_shield"`
			AllSpecialTypes []int `json:"all_special_types"`
			PlayurlInfo     struct {
				ConfJSON string `json:"conf_json"`
				Playurl  struct {
					Cid     int `json:"cid"`
					GQnDesc []struct {
						Qn       int         `json:"qn"`
						Desc     string      `json:"desc"`
						HdrDesc  string      `json:"hdr_desc"`
						AttrDesc interface{} `json:"attr_desc"`
					} `json:"g_qn_desc"`
					Stream []struct {
						ProtocolName string `json:"protocol_name"`
						Format       []struct {
							FormatName string `json:"format_name"`
							Codec      []struct {
								CodecName string `json:"codec_name"`
								CurrentQn int    `json:"current_qn"`
								AcceptQn  []int  `json:"accept_qn"`
								BaseURL   string `json:"base_url"`
								URLInfo   []struct {
									Host      string `json:"host"`
									Extra     string `json:"extra"`
									StreamTTL int    `json:"stream_ttl"`
								} `json:"url_info"`
								HdrQn     interface{} `json:"hdr_qn"`
								DolbyType int         `json:"dolby_type"`
								AttrName  string      `json:"attr_name"`
							} `json:"codec"`
						} `json:"format"`
					} `json:"stream"`
					P2PData struct {
						P2P      bool        `json:"p2p"`
						P2PType  int         `json:"p2p_type"`
						MP2P     bool        `json:"m_p2p"`
						MServers interface{} `json:"m_servers"`
					} `json:"p2p_data"`
					DolbyQn interface{} `json:"dolby_qn"`
				} `json:"playurl"`
			} `json:"playurl_info"`
		} `json:"data"`
	}

	err = json.Unmarshal(req.Respon, &j)
	if err != nil {
		return
	} else if j.Code != 0 {
		err = errors.New(j.Message)
		return
	}

	//主播uid
	res.UpUid = j.Data.UID
	//房间号（完整）
	res.RoomID = j.Data.RoomID
	//直播开始时间
	if j.Data.LiveTime != 0 {
		res.LiveStartTime = time.Unix(int64(j.Data.LiveTime), 0)
	}
	//是否在直播
	res.Liveing = j.Data.LiveStatus == 1

	//当前直播流
	res.Streams = []struct {
		ProtocolName string
		Format       []struct {
			FormatName string
			Codec      []struct {
				CodecName string
				CurrentQn int
				AcceptQn  []int
				BaseURL   string
				URLInfo   []struct {
					Host      string
					Extra     string
					StreamTTL int
				}
				HdrQn     any
				DolbyType int
				AttrName  string
			}
		}
	}(j.Data.PlayurlInfo.Playurl.Stream)
	return
}

// SetCookies implements biliApiInter.
func (t *biliApi) SetCookies(cookies []*http.Cookie) {
	for i := 0; i < len(cookies); i++ {
		found := false
		for k := 0; k < len(t.cookies); k++ {
			if t.cookies[k].Name == cookies[i].Name {
				t.cookies[k].Value = cookies[i].Value
				found = true
				break
			}
		}
		if !found {
			t.cookies = append(t.cookies, cookies[i])
		}
	}
}

// GetInfoByRoom implements biliApiInter.
func (t *biliApi) GetInfoByRoom(Roomid int) (err error, res struct {
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
}) {
	req := t.pool.Get()
	defer t.pool.Put(req)
	err = req.Reqf(reqf.Rval{
		Url: fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/index/getInfoByRoom?room_id=%d", Roomid),
		Header: map[string]string{
			`Referer`: fmt.Sprintf("https://live.bilibili.com/%d", Roomid),
		},
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
		Retry:   2,
	})
	if err != nil {
		return
	}

	//Roominfores
	{
		var j struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			TTL     int    `json:"ttl"`
			Data    struct {
				RoomInfo struct {
					UID            int    `json:"uid"`
					RoomID         int    `json:"room_id"`
					ShortID        int    `json:"short_id"`
					Title          string `json:"title"`
					Cover          string `json:"cover"`
					Tags           string `json:"tags"`
					Background     string `json:"background"`
					Description    string `json:"description"`
					LiveStatus     int    `json:"live_status"`
					LiveStartTime  int    `json:"live_start_time"`
					LiveScreenType int    `json:"live_screen_type"`
					LockStatus     int    `json:"lock_status"`
					LockTime       int    `json:"lock_time"`
					HiddenStatus   int    `json:"hidden_status"`
					HiddenTime     int    `json:"hidden_time"`
					AreaID         int    `json:"area_id"`
					AreaName       string `json:"area_name"`
					ParentAreaID   int    `json:"parent_area_id"`
					ParentAreaName string `json:"parent_area_name"`
					Keyframe       string `json:"keyframe"`
					SpecialType    int    `json:"special_type"`
					UpSession      string `json:"up_session"`
					PkStatus       int    `json:"pk_status"`
					IsStudio       bool   `json:"is_studio"`
					Pendants       struct {
						Frame struct {
							Name  string `json:"name"`
							Value string `json:"value"`
							Desc  string `json:"desc"`
						} `json:"frame"`
					} `json:"pendants"`
					OnVoiceJoin int `json:"on_voice_join"`
					Online      int `json:"online"`
					RoomType    struct {
						Two3    int `json:"2-3"`
						Three21 int `json:"3-21"`
					} `json:"room_type"`
				} `json:"room_info"`
				AnchorInfo struct {
					BaseInfo struct {
						Uname        string `json:"uname"`
						Face         string `json:"face"`
						Gender       string `json:"gender"`
						OfficialInfo struct {
							Role     int    `json:"role"`
							Title    string `json:"title"`
							Desc     string `json:"desc"`
							IsNft    int    `json:"is_nft"`
							NftDmark string `json:"nft_dmark"`
						} `json:"official_info"`
					} `json:"base_info"`
					LiveInfo struct {
						Level        int           `json:"level"`
						LevelColor   int           `json:"level_color"`
						Score        int           `json:"score"`
						UpgradeScore int           `json:"upgrade_score"`
						Current      []int         `json:"current"`
						Next         []interface{} `json:"next"`
						Rank         string        `json:"rank"`
					} `json:"live_info"`
					RelationInfo struct {
						Attention int `json:"attention"`
					} `json:"relation_info"`
					MedalInfo struct {
						MedalName string `json:"medal_name"`
						MedalID   int    `json:"medal_id"`
						Fansclub  int    `json:"fansclub"`
					} `json:"medal_info"`
					GiftInfo interface{} `json:"gift_info"`
				} `json:"anchor_info"`
				NewsInfo struct {
					UID     int    `json:"uid"`
					Ctime   string `json:"ctime"`
					Content string `json:"content"`
				} `json:"news_info"`
				RankdbInfo struct {
					Roomid    int    `json:"roomid"`
					RankDesc  string `json:"rank_desc"`
					Color     string `json:"color"`
					H5URL     string `json:"h5_url"`
					WebURL    string `json:"web_url"`
					Timestamp int    `json:"timestamp"`
				} `json:"rankdb_info"`
				AreaRankInfo struct {
					AreaRank struct {
						Index int    `json:"index"`
						Rank  string `json:"rank"`
					} `json:"areaRank"`
					LiveRank struct {
						Rank string `json:"rank"`
					} `json:"liveRank"`
				} `json:"area_rank_info"`
				BattleRankEntryInfo interface{} `json:"battle_rank_entry_info"`
				TabInfo             struct {
					List []struct {
						Type      string `json:"type"`
						Desc      string `json:"desc"`
						IsFirst   int    `json:"isFirst"`
						IsEvent   int    `json:"isEvent"`
						EventType string `json:"eventType"`
						ListType  string `json:"listType"`
						APIPrefix string `json:"apiPrefix"`
						RankName  string `json:"rank_name"`
					} `json:"list"`
				} `json:"tab_info"`
				ActivityInitInfo struct {
					EventList []interface{} `json:"eventList"`
					WeekInfo  struct {
						BannerInfo interface{} `json:"bannerInfo"`
						GiftName   interface{} `json:"giftName"`
					} `json:"weekInfo"`
					GiftName interface{} `json:"giftName"`
					Lego     struct {
						Timestamp int    `json:"timestamp"`
						Config    string `json:"config"`
					} `json:"lego"`
				} `json:"activity_init_info"`
				VoiceJoinInfo struct {
					Status struct {
						Open        int    `json:"open"`
						AnchorOpen  int    `json:"anchor_open"`
						Status      int    `json:"status"`
						UID         int    `json:"uid"`
						UserName    string `json:"user_name"`
						HeadPic     string `json:"head_pic"`
						Guard       int    `json:"guard"`
						StartAt     int    `json:"start_at"`
						CurrentTime int    `json:"current_time"`
					} `json:"status"`
					Icons struct {
						IconClose    string `json:"icon_close"`
						IconOpen     string `json:"icon_open"`
						IconWait     string `json:"icon_wait"`
						IconStarting string `json:"icon_starting"`
					} `json:"icons"`
					WebShareLink string `json:"web_share_link"`
				} `json:"voice_join_info"`
				AdBannerInfo struct {
					Data []struct {
						ID                   int         `json:"id"`
						Title                string      `json:"title"`
						Location             string      `json:"location"`
						Position             int         `json:"position"`
						Pic                  string      `json:"pic"`
						Link                 string      `json:"link"`
						Weight               int         `json:"weight"`
						RoomID               int         `json:"room_id"`
						UpID                 int         `json:"up_id"`
						ParentAreaID         int         `json:"parent_area_id"`
						AreaID               int         `json:"area_id"`
						LiveStatus           int         `json:"live_status"`
						AvID                 int         `json:"av_id"`
						IsAd                 bool        `json:"is_ad"`
						AdTransparentContent interface{} `json:"ad_transparent_content"`
						ShowAdIcon           bool        `json:"show_ad_icon"`
					} `json:"data"`
				} `json:"ad_banner_info"`
				SkinInfo struct {
					ID          int    `json:"id"`
					SkinName    string `json:"skin_name"`
					SkinConfig  string `json:"skin_config"`
					ShowText    string `json:"show_text"`
					SkinURL     string `json:"skin_url"`
					StartTime   int    `json:"start_time"`
					EndTime     int    `json:"end_time"`
					CurrentTime int    `json:"current_time"`
				} `json:"skin_info"`
				WebBannerInfo struct {
					ID               int    `json:"id"`
					Title            string `json:"title"`
					Left             string `json:"left"`
					Right            string `json:"right"`
					JumpURL          string `json:"jump_url"`
					BgColor          string `json:"bg_color"`
					HoverColor       string `json:"hover_color"`
					TextBgColor      string `json:"text_bg_color"`
					TextHoverColor   string `json:"text_hover_color"`
					LinkText         string `json:"link_text"`
					LinkColor        string `json:"link_color"`
					InputColor       string `json:"input_color"`
					InputTextColor   string `json:"input_text_color"`
					InputHoverColor  string `json:"input_hover_color"`
					InputBorderColor string `json:"input_border_color"`
					InputSearchColor string `json:"input_search_color"`
				} `json:"web_banner_info"`
				LolInfo        interface{} `json:"lol_info"`
				PkInfo         interface{} `json:"pk_info"`
				BattleInfo     interface{} `json:"battle_info"`
				SilentRoomInfo struct {
					Type       string `json:"type"`
					Level      int    `json:"level"`
					Second     int    `json:"second"`
					ExpireTime int    `json:"expire_time"`
				} `json:"silent_room_info"`
				SwitchInfo struct {
					CloseGuard   bool `json:"close_guard"`
					CloseGift    bool `json:"close_gift"`
					CloseOnline  bool `json:"close_online"`
					CloseDanmaku bool `json:"close_danmaku"`
				} `json:"switch_info"`
				RecordSwitchInfo interface{} `json:"record_switch_info"`
				RoomConfigInfo   struct {
					DmText string `json:"dm_text"`
				} `json:"room_config_info"`
				GiftMemoryInfo struct {
					List interface{} `json:"list"`
				} `json:"gift_memory_info"`
				NewSwitchInfo struct {
					RoomSocket           int `json:"room-socket"`
					RoomPropSend         int `json:"room-prop-send"`
					RoomSailing          int `json:"room-sailing"`
					RoomInfoPopularity   int `json:"room-info-popularity"`
					RoomDanmakuEditor    int `json:"room-danmaku-editor"`
					RoomEffect           int `json:"room-effect"`
					RoomFansMedal        int `json:"room-fans_medal"`
					RoomReport           int `json:"room-report"`
					RoomFeedback         int `json:"room-feedback"`
					RoomPlayerWatermark  int `json:"room-player-watermark"`
					RoomRecommendLiveOff int `json:"room-recommend-live_off"`
					RoomActivity         int `json:"room-activity"`
					RoomWebBanner        int `json:"room-web_banner"`
					RoomSilverSeedsBox   int `json:"room-silver_seeds-box"`
					RoomWishingBottle    int `json:"room-wishing_bottle"`
					RoomBoard            int `json:"room-board"`
					RoomSupplication     int `json:"room-supplication"`
					RoomHourRank         int `json:"room-hour_rank"`
					RoomWeekRank         int `json:"room-week_rank"`
					RoomAnchorRank       int `json:"room-anchor_rank"`
					RoomInfoIntegral     int `json:"room-info-integral"`
					RoomSuperChat        int `json:"room-super-chat"`
					RoomTab              int `json:"room-tab"`
					RoomHotRank          int `json:"room-hot-rank"`
					FansMedalProgress    int `json:"fans-medal-progress"`
					GiftBayScreen        int `json:"gift-bay-screen"`
					RoomEnter            int `json:"room-enter"`
					RoomMyIdol           int `json:"room-my-idol"`
					RoomTopic            int `json:"room-topic"`
					FansClub             int `json:"fans-club"`
					RoomPopularRank      int `json:"room-popular-rank"`
					MicUserGift          int `json:"mic_user_gift"`
					NewRoomAreaRank      int `json:"new-room-area-rank"`
				} `json:"new_switch_info"`
				SuperChatInfo struct {
					Status      int           `json:"status"`
					JumpURL     string        `json:"jump_url"`
					Icon        string        `json:"icon"`
					RankedMark  int           `json:"ranked_mark"`
					MessageList []interface{} `json:"message_list"`
				} `json:"super_chat_info"`
				OnlineGoldRankInfoV2 struct {
					List []struct {
						UID        int64  `json:"uid"`
						Face       string `json:"face"`
						Uname      string `json:"uname"`
						Score      string `json:"score"`
						Rank       int    `json:"rank"`
						GuardLevel int    `json:"guard_level"`
					} `json:"list"`
				} `json:"online_gold_rank_info_v2"`
				DmBrushInfo struct {
					MinTime     int `json:"min_time"`
					BrushCount  int `json:"brush_count"`
					SliceCount  int `json:"slice_count"`
					StorageTime int `json:"storage_time"`
				} `json:"dm_brush_info"`
				DmEmoticonInfo struct {
					IsOpenEmoticon   int `json:"is_open_emoticon"`
					IsShieldEmoticon int `json:"is_shield_emoticon"`
				} `json:"dm_emoticon_info"`
				DmTagInfo struct {
					DmTag           int           `json:"dm_tag"`
					Platform        []interface{} `json:"platform"`
					Extra           string        `json:"extra"`
					DmChronosExtra  string        `json:"dm_chronos_extra"`
					DmMode          []interface{} `json:"dm_mode"`
					DmSettingSwitch int           `json:"dm_setting_switch"`
					MaterialConf    interface{}   `json:"material_conf"`
				} `json:"dm_tag_info"`
				TopicInfo struct {
					TopicID   int    `json:"topic_id"`
					TopicName string `json:"topic_name"`
				} `json:"topic_info"`
				GameInfo struct {
					GameStatus int `json:"game_status"`
				} `json:"game_info"`
				WatchedShow struct {
					Switch       bool   `json:"switch"`
					Num          int    `json:"num"`
					TextSmall    string `json:"text_small"`
					TextLarge    string `json:"text_large"`
					Icon         string `json:"icon"`
					IconLocation int    `json:"icon_location"`
					IconWeb      string `json:"icon_web"`
				} `json:"watched_show"`
				TopicRoomInfo struct {
					InteractiveH5URL string `json:"interactive_h5_url"`
					Watermark        int    `json:"watermark"`
				} `json:"topic_room_info"`
				ShowReserveStatus bool `json:"show_reserve_status"`
				SecondCreateInfo  struct {
					ClickPermission  int    `json:"click_permission"`
					CommonPermission int    `json:"common_permission"`
					IconName         string `json:"icon_name"`
					IconURL          string `json:"icon_url"`
					URL              string `json:"url"`
				} `json:"second_create_info"`
				PlayTogetherInfo struct {
					Switch   int `json:"switch"`
					IconList []struct {
						Icon    string `json:"icon"`
						Title   string `json:"title"`
						JumpURL string `json:"jump_url"`
						Status  int    `json:"status"`
					} `json:"icon_list"`
				} `json:"play_together_info"`
				CloudGameInfo struct {
					IsGaming int `json:"is_gaming"`
				} `json:"cloud_game_info"`
				LikeInfoV3 struct {
					TotalLikes    int      `json:"total_likes"`
					ClickBlock    bool     `json:"click_block"`
					CountBlock    bool     `json:"count_block"`
					GuildEmoText  string   `json:"guild_emo_text"`
					GuildDmText   string   `json:"guild_dm_text"`
					LikeDmText    string   `json:"like_dm_text"`
					HandIcons     []string `json:"hand_icons"`
					DmIcons       []string `json:"dm_icons"`
					EggshellsIcon string   `json:"eggshells_icon"`
					CountShowTime int      `json:"count_show_time"`
					ProcessIcon   string   `json:"process_icon"`
					ProcessColor  string   `json:"process_color"`
				} `json:"like_info_v3"`
				LivePlayInfo struct {
					ShowWidgetBanner bool `json:"show_widget_banner"`
				} `json:"live_play_info"`
				MultiVoice struct {
					SwitchStatus int           `json:"switch_status"`
					Members      []interface{} `json:"members"`
				} `json:"multi_voice"`
				PopularRankInfo struct {
					Rank       int    `json:"rank"`
					Countdown  int    `json:"countdown"`
					Timestamp  int    `json:"timestamp"`
					URL        string `json:"url"`
					OnRankName string `json:"on_rank_name"`
					RankName   string `json:"rank_name"`
				} `json:"popular_rank_info"`
				NewAreaRankInfo struct {
					Items []struct {
						ConfID      int    `json:"conf_id"`
						RankName    string `json:"rank_name"`
						UID         int    `json:"uid"`
						Rank        int    `json:"rank"`
						IconURLBlue string `json:"icon_url_blue"`
						IconURLPink string `json:"icon_url_pink"`
						IconURLGrey string `json:"icon_url_grey"`
						JumpURLLink string `json:"jump_url_link"`
						JumpURLPc   string `json:"jump_url_pc"`
						JumpURLPink string `json:"jump_url_pink"`
						JumpURLWeb  string `json:"jump_url_web"`
					} `json:"items"`
					RotationCycleTimeWeb int `json:"rotation_cycle_time_web"`
				} `json:"new_area_rank_info"`
				GiftStar struct {
					Show bool `json:"show"`
				} `json:"gift_star"`
				VideoConnectionInfo interface{} `json:"video_connection_info"`
				PlayerThrottleInfo  struct {
					Status              int `json:"status"`
					NormalSleepTime     int `json:"normal_sleep_time"`
					FullscreenSleepTime int `json:"fullscreen_sleep_time"`
					TabSleepTime        int `json:"tab_sleep_time"`
					PromptTime          int `json:"prompt_time"`
				} `json:"player_throttle_info"`
				GuardInfo struct {
					Count                   int `json:"count"`
					AnchorGuardAchieveLevel int `json:"anchor_guard_achieve_level"`
				} `json:"guard_info"`
				HotRankInfo interface{} `json:"hot_rank_info"`
			} `json:"data"`
		}

		err = json.Unmarshal(req.Respon, &j)
		if err != nil {
			return
		} else if j.Code != 0 {
			err = errors.New(j.Message)
			return
		}

		//直播开始时间
		if j.Data.RoomInfo.LiveStartTime != 0 {
			res.LiveStartTime = time.Unix(int64(j.Data.RoomInfo.LiveStartTime), 0)
		}
		//是否在直播
		res.Liveing = j.Data.RoomInfo.LiveStatus == 1
		//直播间标题
		res.Title = j.Data.RoomInfo.Title
		//主播名
		res.Uname = j.Data.AnchorInfo.BaseInfo.Uname
		//分区
		res.ParentAreaID = j.Data.RoomInfo.ParentAreaID
		//子分区
		res.AreaID = j.Data.RoomInfo.AreaID
		//主播id
		res.UpUid = j.Data.RoomInfo.UID
		//房间id
		res.RoomID = j.Data.RoomInfo.RoomID
		//舰长数
		res.GuardNum = j.Data.GuardInfo.Count
		//分区排行
		res.Note = j.Data.PopularRankInfo.RankName + " "
		if rank := j.Data.PopularRankInfo.Rank; rank > 50 || rank == 0 {
			res.Note += "100+"
		} else {
			res.Note += strconv.Itoa(rank)
		}
		//直播间是否被封禁
		res.Locked = j.Data.RoomInfo.LockStatus == 1
	}
	return
}

func (t *biliApi) SetReqPool(pool *pool.Buf[reqf.Req]) {
	t.pool = pool
}

// GetRoomBaseInfo implements biliApiInter.
func (t *biliApi) GetRoomBaseInfo(Roomid int) (err error, res struct {
	UpUid         int
	Uname         string
	ParentAreaID  int
	AreaID        int
	Title         string
	LiveStartTime time.Time
	Liveing       bool
	RoomID        int
}) {
	req := t.pool.Get()
	defer t.pool.Put(req)

	err = req.Reqf(reqf.Rval{
		Url: fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/index/getRoomBaseInfo?req_biz=link-center&room_ids=%d", Roomid),
		Header: map[string]string{
			`Referer`: "https://link.bilibili.com/p/center/index",
		},
		Proxy:   t.proxy,
		Timeout: 10 * 1000,
	})
	if err != nil {
		return
	}

	//Roominfores
	{
		var j struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			TTL     int    `json:"ttl"`
			Data    struct {
				ByRoomIds map[string]struct {
					RoomID     int `json:"room_id"`
					UID        int `json:"uid"`
					AreaID     int `json:"area_id"`
					LiveStatus int `json:"live_status"`
					// LiveURL        string `json:"live_url"`
					ParentAreaID int    `json:"parent_area_id"`
					Title        string `json:"title"`
					// ParentAreaName string `json:"parent_area_name"`
					// AreaName       string `json:"area_name"`
					LiveTime string `json:"live_time"`
					// Description    string `json:"description"`
					// Tags           string `json:"tags"`
					Attention  int    `json:"attention"`
					Online     int    `json:"online"`
					ShortID    int    `json:"short_id"`
					Uname      string `json:"uname"`
					Cover      string `json:"cover"`
					Background string `json:"background"`
					JoinSlide  int    `json:"join_slide"`
					LiveID     int64  `json:"live_id"`
					LiveIDStr  string `json:"live_id_str"`
				} `json:"by_room_ids"`
			} `json:"data"`
		}

		err = json.Unmarshal(req.Respon, &j)
		if err != nil {
			return
		} else if j.Code != 0 {
			err = errors.New(j.Message)
			return
		}

		for _, data := range j.Data.ByRoomIds {
			if Roomid == data.RoomID || Roomid == data.ShortID {
				//主播id
				res.UpUid = data.UID
				//子分区
				res.AreaID = data.AreaID
				//分区
				res.ParentAreaID = data.ParentAreaID
				//直播间标题
				res.Title = data.Title
				//直播开始时间
				if ti, e := time.Parse(time.DateTime, data.LiveTime); e != nil && !ti.IsZero() {
					res.LiveStartTime = ti
				}
				//是否在直播
				res.Liveing = data.LiveStatus == 1
				//主播名
				res.Uname = data.Uname
				//房间id
				res.RoomID = data.RoomID
				return
			}
		}
	}
	return
}

// LoginQrPoll implements F.BiliApi.
func (t *biliApi) LoginQrPoll(QrcodeKey string) (err error, cookies []*http.Cookie) {
	r := t.pool.Get()
	defer t.pool.Put(r)
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

func (t *biliApi) SetProxy(proxy string) {
	t.proxy = proxy
}

func (t *biliApi) LoginQrCode() (err error, imgUrl string, QrcodeKey string) {
	r := t.pool.Get()
	defer t.pool.Put(r)
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

// bilibili wrid wts 计算
func getWridWts(query string, imgURL, subURL string, customWts ...string) (w_rid, wts string) {
	wbi := imgURL[strings.LastIndex(imgURL, "/")+1:strings.LastIndex(imgURL, ".")] +
		subURL[strings.LastIndex(subURL, "/")+1:strings.LastIndex(subURL, ".")]

	code := []int{46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5,
		49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55,
		40, 61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57,
		62, 11, 36, 20, 34, 44, 52}

	s := []byte{}

	for i := 0; i < len(code); i++ {
		if code[i] < len(wbi) {
			s = append(s, wbi[code[i]])
			if len(s) >= 32 {
				break
			}
		}
	}

	object := strings.Split(query, "&")

	if len(customWts) == 0 {
		wts = fmt.Sprintf("%d", time.Now().Unix())
	} else {
		wts = customWts[0]
	}
	object = append(object, "wts="+wts)

	slices.Sort(object)

	for i := 0; i < len(object); i++ {
		object[i] = url.PathEscape(object[i])
	}

	w_rid = fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(object, "&")+string(s))))

	return
}
