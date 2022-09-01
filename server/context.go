package server

type Event struct {
	Channel   *Channel
	Conn      *Connection
	Msg       *ClientMessage
	dataCache interface{}
}

func NewEvent(channel *Channel, conn *Connection, msg *ClientMessage) *Event {
	return &Event{
		Channel: channel,
		Conn:    conn,
		Msg:     msg,
	}
}

func (c *Event) Param(p string) string {
	return c.Channel.Params.Param(p)
}

func (c *Event) Name() string {
	return c.Msg.Event
}

func (c *Event) Type() ConnectionEvent {
	return c.Msg.Type
}

func (c *Event) Data(v interface{}) error {
	if c.dataCache != nil {
		return nil
	}

	err := c.Msg.Data(v)

	c.dataCache = v

	return err
}

func (c *Event) Broadcast(event string, data interface{}) {
	c.Channel.Broadcast(event, data, c.Conn)
}

func (c *Event) Emit(event string, data interface{}) {
	c.Channel.Emit(event, data)
}

func (c *Event) Send(event string, data interface{}) {
	serverMessage := c.Channel.newServerMessage(event, data)
	bytes, err := serverMessage.Marshal()

	if err != nil {
		return
	}

	c.Conn.byteSend <- bytes
}

func (c *Event) Ack(data interface{}) {
	serverMessage := c.Channel.newServerMessage("ack", data)

	bytes, err := serverMessage.Marshal()

	if err != nil {
		return
	}

	c.Conn.byteSend <- bytes
}
