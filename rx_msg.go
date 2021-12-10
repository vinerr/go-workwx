package workwx

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// RxMessage 一条接收到的消息
type RxMessage struct {
	CorpID     string      // 接收消息企业ID
	FromUserID string      // FromUserID 发送者的 UserID
	SendTime   time.Time   // SendTime 消息发送时间
	MsgType    MessageType // MsgType 消息类型
	MsgID      int64       // MsgID 消息 ID
	AgentID    int64       // AgentID 企业应用 ID，可在应用的设置页面查看
	Event      EventType   // Event 事件类型 MsgType为event存在
	ChangeType ChangeType  // ChangeType 变更类型 Event为change_external_contact存在
	extras     messageKind
}

func fromEnvelope(body []byte) (*RxMessage, error) {
	// extract common part
	var common rxMessageCommon
	err := xml.Unmarshal(body, &common)
	if err != nil {
		return nil, err
	}

	// deal with polymorphic message types
	extras, err := extractMessageExtras(common, body)
	if err != nil {
		return nil, err
	}
	logrus.Debugln(">>>>>>002")

	// assemble message object
	var obj RxMessage
	{
		// let's force people to think about timezones okay?
		// -- let's not
		sendTime := time.Unix(common.CreateTime, 0) // in time.Local

		obj = RxMessage{
			CorpID:     common.ToUserName,
			FromUserID: common.FromUserName,
			SendTime:   sendTime,
			MsgType:    common.MsgType,
			MsgID:      common.MsgID,
			AgentID:    common.AgentID,
			Event:      common.Event,
			ChangeType: common.ChangeType,

			extras: extras,
		}
	}

	return &obj, nil
}

func (m *RxMessage) String() string {
	var sb strings.Builder

	_, _ = fmt.Fprintf(
		&sb,
		"RxMessage { CorpID: %#v, FromUserID: %#v, SendTime: %d, MsgType: %#v, MsgID: %d, AgentID: %d, Event: %#v, ChangeType: %#v, ",
		m.CorpID,
		m.FromUserID,
		m.SendTime.UnixNano(),
		m.MsgType,
		m.MsgID,
		m.AgentID,
		m.Event,
		m.ChangeType,
	)

	m.extras.formatInto(&sb)

	sb.WriteString(" }")

	return sb.String()
}

// Text 如果消息为文本类型，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) Text() (TextMessageExtras, bool) {
	y, ok := m.extras.(TextMessageExtras)
	return y, ok
}

// Image 如果消息为图片类型，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) Image() (ImageMessageExtras, bool) {
	y, ok := m.extras.(ImageMessageExtras)
	return y, ok
}

// Voice 如果消息为语音类型，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) Voice() (VoiceMessageExtras, bool) {
	y, ok := m.extras.(VoiceMessageExtras)
	return y, ok
}

// Video 如果消息为视频类型，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) Video() (VideoMessageExtras, bool) {
	y, ok := m.extras.(VideoMessageExtras)
	return y, ok
}

// Location 如果消息为位置类型，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) Location() (LocationMessageExtras, bool) {
	y, ok := m.extras.(LocationMessageExtras)
	return y, ok
}

// Link 如果消息为链接类型，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) Link() (LinkMessageExtras, bool) {
	y, ok := m.extras.(LinkMessageExtras)
	return y, ok
}

// EventAddExternalContact 如果消息为添加企业客户事件，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventAddExternalContact() (EventAddExternalContact, bool) {
	y, ok := m.extras.(EventAddExternalContact)
	return y, ok
}

// EventEditExternalContact 如果消息为编辑企业客户事件，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventEditExternalContact() (EventEditExternalContact, bool) {
	y, ok := m.extras.(EventEditExternalContact)
	return y, ok
}

// EventDelExternalContact 如果消息为删除企业客户事件，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventDelExternalContact() (EventDelExternalContact, bool) {
	y, ok := m.extras.(EventDelExternalContact)
	return y, ok
}

// EventDelFollowUser 如果消息为删除跟进成员事件，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventDelFollowUser() (EventDelFollowUser, bool) {
	y, ok := m.extras.(EventDelFollowUser)
	return y, ok
}

// EventAddHalfExternalContact 如果消息为外部联系人免验证添加成员事件，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventAddHalfExternalContact() (EventAddHalfExternalContact, bool) {
	y, ok := m.extras.(EventAddHalfExternalContact)
	return y, ok
}

// EventTransferFail 如果消息为客户接替失败事件，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventTransferFail() (EventTransferFail, bool) {
	y, ok := m.extras.(EventTransferFail)
	return y, ok
}

// EventChangeExternalChat 如果消息为客户群变更事件，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventChangeExternalChat() (EventChangeExternalChat, bool) {
	y, ok := m.extras.(EventChangeExternalChat)
	return y, ok
}

// EventSysApprovalChange 如果消息为审批申请状态变化回调通知，则拿出相应的消息参数，否则返回 nil, false
func (m *RxMessage) EventSysApprovalChange() (EventSysApprovalChange, bool) {
	y, ok := m.extras.(EventSysApprovalChange)
	return y, ok
}
