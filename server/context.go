package server

type Context struct {
	Channel   *internalChannel
	Conn      *Connection
	Msg       *ConnectionMessage
	DataCache interface{}
}

func NewContext(channel *internalChannel, conn *Connection, msg *ConnectionMessage) *Context {
	return &Context{
		Channel: channel,
		Conn:    conn,
		Msg:     msg,
	}
}

func (c *Context) Params(p string) string {
	return c.Channel.Params.Param(p)
}

func (c *Context) Event() string {
	return c.Msg.Event
}

func (c *Context) Type() ConnectionEvent {
	return c.Msg.Type
}

func (c *Context) Data(v interface{}) error {
	if c.DataCache != nil {
		return nil
	}

	err := c.Msg.Data(v)

	c.DataCache = v

	return err
}

func (c *Context) Broadcast(event string, data interface{}) {
	c.Channel.Broadcast(event, data, c.Conn)
}

func (c *Context) Emit(event string, data interface{}) {
	c.Channel.Emit(event, data)
}

func (c *Context) Send(event string, data interface{}) {
	serverMessage := c.Channel.newServerMessage(event, data)
	bytes, err := serverMessage.Marshal()

	if err != nil {
		return
	}

	c.Conn.byteSend <- bytes
}
