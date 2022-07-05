package snap

import "github.com/umbracle/go-devp2p"

type SnapProtocol struct {
}

func (s *SnapProtocol) Eth66() *devp2p.Protocol {
	return &devp2p.Protocol{
		Spec: devp2p.ProtocolSpec{
			Name:    "snap",
			Version: 1,
			Length:  8,
		},
		HandlerFn: func(conn1 devp2p.Stream, peer *devp2p.Peer) error {
			return nil
		},
	}
}
