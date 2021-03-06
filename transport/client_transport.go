package transport

import (
	"context"
	"github.com/lubanproj/gorpc/codec"
	"github.com/lubanproj/gorpc/codes"
	"net"
)

type clientTransport struct {
	opts *ClientTransportOptions
}

var clientTransportMap = make(map[string]ClientTransport)

func init() {
	clientTransportMap["default"] = DefaultClientTransport
}

func GetClientTransport(transport string) ClientTransport {

	if v, ok := clientTransportMap[transport]; ok {
		return v
	}

	return DefaultClientTransport
}

var DefaultClientTransport = New()

var New = func() ClientTransport {
	return &clientTransport{
		opts : &ClientTransportOptions{},
	}
}

func (c *clientTransport) Send(ctx context.Context, req []byte, opts ...ClientTransportOption) ([]byte, error) {

	for _, o := range opts {
		o(c.opts)
	}

	if c.opts.Network == "tcp" {
		return c.SendTcpReq(ctx, req)
	}

	if c.opts.Network == "udp" {
		return c.SendUdpReq(ctx, req)
	}

	return nil, codes.NetworkNotSupportedError
}

func (c *clientTransport) SendTcpReq(ctx context.Context, req []byte) ([]byte, error) {

	// service discovery
	addr, err := c.opts.Selector.Select(c.opts.ServiceName)
	if err != nil {
		return nil, err
	}

	// defaultSelector returns "", use the target as address
	if addr == "" {
		addr = c.opts.Target
	}

//	conn, err := c.opts.Pool.Get(ctx, "tcp", addr)
	conn, err := net.Dial("tcp", addr);
	if err != nil {
		return nil, codes.ConnectionError
	}
	defer conn.Close()

	sendNum := 0
	num := 0
	for sendNum < len(req) {
		num , err = conn.Write(req)
		if err != nil {
			return nil, codes.NewFrameworkError(codes.ClientNetworkErrorCode,err.Error())
		}
		sendNum += num

		if err = isDone(ctx); err != nil {
			return nil, err
		}
	}

	// parse frame
	frame, err := codec.ReadFrame(conn)
	if err != nil {
		return nil, codes.NewFrameworkError(codes.ClientNetworkErrorCode, err.Error())
	}

	return frame, err
}

func (c *clientTransport) SendUdpReq(ctx context.Context, req []byte) ([]byte, error) {

	return nil, nil
}


func isDone(ctx context.Context) error {
	select {
	case <- ctx.Done() :
		if ctx.Err() == context.Canceled {
			return codes.ClientContextCanceledError
		}
		if ctx.Err() == context.DeadlineExceeded {
			return codes.ClientTimeoutError
		}
	default:
	}

	return nil
}