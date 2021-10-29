package dtmcli

import (
	"errors"
	"fmt"
	"net/url"
)

// M a short name
type M = map[string]interface{}

// MS a short name
type MS = map[string]string

// MustGenGid generate a new gid
func MustGenGid(server string) string {
	res := MS{}
	resp, err := RestyClient.R().SetResult(&res).Get(server + "/newGid")
	if err != nil || res["gid"] == "" {
		panic(fmt.Errorf("newGid error: %v, resp: %s", err, resp))
	}
	return res["gid"]
}

// IDGenerator used to generate a branch id
type IDGenerator struct {
	parentID string
	branchID int
}

// NewBranchID generate a branch id
func (g *IDGenerator) NewBranchID() string {
	if g.branchID >= 99 {
		panic(fmt.Errorf("branch id is larger than 99"))
	}
	if len(g.parentID) >= 20 {
		panic(fmt.Errorf("total branch id is longer than 20"))
	}
	g.branchID = g.branchID + 1
	return g.CurrentBranchID()
}

// CurrentBranchID return current branchID
func (g *IDGenerator) CurrentBranchID() string {
	return g.parentID + fmt.Sprintf("%02d", g.branchID)
}

// TransResult dtm 返回的结果
type TransResult struct {
	DtmResult string `json:"dtm_result"`
	Message   string
}

// TransOptions transaction options
type TransOptions struct {
	WaitResult    bool  `json:"wait_result,omitempty" gorm:"-"`
	TimeoutToFail int64 `json:"timeout_to_fail,omitempty" gorm:"-"` // for trans type: xa, tcc
	RetryInterval int64 `json:"retry_interval,omitempty" gorm:"-"`  // for trans type: msg saga xa tcc
}

// TransBase 事务的基础类
type TransBase struct {
	Gid        string `json:"gid"`
	TransType  string `json:"trans_type"`
	Dtm        string `json:"-"`
	CustomData string `json:"custom_data,omitempty"`
	IDGenerator
	TransOptions
}

// SetOptions set options
func (tb *TransBase) SetOptions(options *TransOptions) {
	tb.TransOptions = *options
}

// NewTransBase 1
func NewTransBase(gid string, transType string, dtm string, parentID string) *TransBase {
	return &TransBase{
		Gid:         gid,
		TransType:   transType,
		IDGenerator: IDGenerator{parentID: parentID},
		Dtm:         dtm,
	}
}

// TransBaseFromQuery construct transaction info from request
func TransBaseFromQuery(qs url.Values) *TransBase {
	return NewTransBase(qs.Get("gid"), qs.Get("trans_type"), qs.Get("dtm"), qs.Get("branch_id"))
}

// callDtm 调用dtm服务器，返回事务的状态
func (tb *TransBase) callDtm(body interface{}, operation string) error {
	resp, err := RestyClient.R().
		SetResult(&TransResult{}).SetBody(body).Post(fmt.Sprintf("%s/%s", tb.Dtm, operation))
	if err != nil {
		return err
	}
	tr := resp.Result().(*TransResult)
	if tr.DtmResult == ResultFailure {
		return errors.New("FAILURE: " + tr.Message)
	}
	return nil
}

// ErrFailure 表示返回失败，要求回滚
var ErrFailure = errors.New("FAILURE")

// ErrOngoing 表示暂时失败，要求重试
var ErrOngoing = errors.New("ONGOING")

// MapSuccess 表示返回成功，可以进行下一步
var MapSuccess = M{"dtm_result": ResultSuccess}

// MapFailure 表示返回失败，要求回滚
var MapFailure = M{"dtm_result": ResultFailure}
