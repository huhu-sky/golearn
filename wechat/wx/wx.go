package wx

import (
	"net/http"
	"io/ioutil"
	"sort"
	"crypto/sha1"
	"fmt"
	"errors"
	"log"
	"encoding/xml"

	"github.com/clbanning/mxj"
)

type weixinQuery struct {
	Signature		string `json:"signature"`
	Timestamp		string  `json:"Timestamp"`
	Nonce			string `json:"Nonce"`
	EncryptType		string `json:"EncryptType"`
	MsgSignature	string `json:"MsgSignature"`
	Echostr			string `json:"Echostr"`

}

type WeixinClient struct {
	Token			string
	Query			weixinQuery
	Message			map[string]interface{}
	Request			*http.Request
	ResponseWriter	http.ResponseWriter
	Methods			map[string]func() bool
}

func (this *WeixinClient) initWeixinQuery() {
	var q weixinQuery
	q.Nonce = this.Request.URL.Query().Get("nonce")
	q.Echostr = this.Request.URL.Query().Get("echostr")
	q.Signature = this.Request.URL.Query().Get("signature")
	q.Timestamp = this.Request.URL.Query().Get("timestamp")
	q.EncryptType = this.Request.URL.Query().Get("encrypt_type")
	q.MsgSignature = this.Request.URL.Query().Get("msg_signature")
	this.Query = q
}

func NewClient(r *http.Request, w http.ResponseWriter, token string) (*WeixinClient, error)  {
	WeixinClient := new(WeixinClient)
	WeixinClient.Token = token
	WeixinClient.Request = r
	WeixinClient.ResponseWriter = w

	WeixinClient.initWeixinQuery()

	return WeixinClient, nil
}

func (this *WeixinClient) signature() string {
	strs := sort.StringSlice{this.Token, this.Query.Timestamp, this.Query.Nonce}
	sort.Strings(strs)
	str := ""
	for _, s := range strs {
		str += s
	}
	h := sha1.New()
	h.Write([]byte(str))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (this *WeixinClient) initMessage() error {
	body, err := ioutil.ReadAll(this.Request.Body)
	if err != nil {
		return err
	}

	m, err := mxj.NewMapXml(body)

	if err != nil {
		return err
	}

	if _, ok := m["xml"]; !ok {
		return errors.New("Invalid Message")
	}
	message, ok := m["xml"].(map[string]interface{})

	if !ok {
		return errors.New("Invalid Field `xml` Type")
	}

	this.Message = message

	log.Println(this.Message)

	return nil

}

func (this *WeixinClient) text() {
	inMsg, ok := this.Message["Content"].(string)
	if !ok {
		return
	}

	var reply TextMessage

	reply.InitBaseData(this, "text")
	reply.Content = value2CDATA(fmt.Sprintf("我收到的是： %s", inMsg))

	replyXml, err := xml.Marshal(reply)
	if err != nil {
		log.Println(err)
		this.ResponseWriter.WriteHeader(403)
		return
	}

	this.ResponseWriter.Header().Set("Content-Type", "text/xml")
	this.ResponseWriter.Write(replyXml)
}

func (this *WeixinClient) Run() {

	err := this.initMessage()

	if err != nil {
		log.Println(err)
		this.ResponseWriter.WriteHeader(403)
		return
	}

	MsgType, ok := this.Message["MsgType"].(string)

	if !ok {
		this.ResponseWriter.WriteHeader(403)
		return
	}

	switch MsgType {
	case "text":
		this.text()
		break
	default:
		break
	}

	return

}