package websocket

import (
	"code.google.com/p/gogoprotobuf/proto"
	"doppler/sinks"
	"github.com/cloudfoundry/dropsonde/events"
	"github.com/cloudfoundry/gosteno"
	gorilla "github.com/gorilla/websocket"
	"net"
)

const FIREHOSE_APP_ID = "firehose"

type remoteMessageWriter interface {
	RemoteAddr() net.Addr
	WriteMessage(messageType int, data []byte) error
}

type WebsocketSink struct {
	logger              *gosteno.Logger
	appId               string
	ws                  remoteMessageWriter
	clientAddress       net.Addr
	wsMessageBufferSize uint
	dropsondeOrigin     string
}

func NewWebsocketSink(appId string, givenLogger *gosteno.Logger, ws remoteMessageWriter, wsMessageBufferSize uint, dropsondeOrigin string) *WebsocketSink {
	return &WebsocketSink{
		logger:              givenLogger,
		appId:               appId,
		ws:                  ws,
		clientAddress:       ws.RemoteAddr(),
		wsMessageBufferSize: wsMessageBufferSize,
		dropsondeOrigin:     dropsondeOrigin,
	}
}

func (sink *WebsocketSink) Identifier() string {
	return sink.ws.RemoteAddr().String()
}

func (sink *WebsocketSink) AppId() string {
	return sink.appId
}

func (sink *WebsocketSink) ShouldReceiveErrors() bool {
	return true
}

func (sink *WebsocketSink) Run(inputChan <-chan *events.Envelope) {
	sink.logger.Debugf("Websocket Sink %s: Running for appId [%s]", sink.clientAddress, sink.appId)

	buffer := sinks.RunTruncatingBuffer(inputChan, sink.wsMessageBufferSize, sink.logger, sink.dropsondeOrigin)
	for {
		sink.logger.Debugf("Websocket Sink %s: Waiting for activity", sink.clientAddress)
		messageEnvelope, ok := <-buffer.GetOutputChannel()
		if !ok {
			sink.logger.Debugf("Websocket Sink %s: Closed listener channel detected. Closing websocket", sink.clientAddress)
			return
		}

		messageBytes, err := proto.Marshal(messageEnvelope)

		if err != nil {
			sink.logger.Errorf("Websocket Sink %s: Error marshalling %s envelope from origin %s: %s", sink.clientAddress, messageEnvelope.GetEventType().String(), messageEnvelope.GetOrigin(), err.Error())
			continue
		}

		sink.logger.Debugf("Websocket Sink %s: Received %s message from %s at %d. Sending data.", sink.clientAddress, messageEnvelope.GetEventType().String(), messageEnvelope.Origin, messageEnvelope.Timestamp)

		err = sink.ws.WriteMessage(gorilla.BinaryMessage, messageBytes)
		if err != nil {
			sink.logger.Debugf("Websocket Sink %s: Error when trying to send data to sink %s. Requesting close. Err: %v", sink.clientAddress, err)
			return
		}

		sink.logger.Debugf("Websocket Sink %s: Successfully sent data", sink.clientAddress)
	}
}
